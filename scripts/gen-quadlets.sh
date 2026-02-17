#!/bin/sh
set -e

# The script generates Quadlet configs from Docker Compose configs.
# Podman processes are booted from Quadlet configs as Systemd services.
#
# Properties:
# Agnostic: User (%h), Kernel (:Z)
# Decoupled: Config (/etc/project/.env)

# --- Configuration ---
# Use readlink to handle symlinks (crucial for Fedora/Silverblue /var/home)
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
REAL_PROJECT_ROOT=$(readlink -f "$PROJECT_ROOT")
PROJECT_DIR_NAME=$(basename "$REAL_PROJECT_ROOT")

# EXTRA DIRECTORY NAMES (constants)
INPUT_DIR_NAME="docker-compose-ymls"
OUTPUT_DIR_NAME="podman-quadlets"
QUADLET_DIR_NAME=".config/containers/systemd"

# ABSOLUTE PATHS (for script execution)
INPUT_DIR="${REAL_PROJECT_ROOT}/${INPUT_DIR_NAME}"
OUTPUT_DIR="${REAL_PROJECT_ROOT}/${OUTPUT_DIR_NAME}"
QUADLET_DIR="${HOME}/${QUADLET_DIR_NAME}"

# SHORT PATHS (for logging)
OUTPUT_DIR_SHORT="${PROJECT_DIR_NAME}/${OUTPUT_DIR_NAME}"
QUADLET_DIR_SHORT="~/${QUADLET_DIR_NAME}"

# --- USER-AGNOSTIC PATHS (Fixed for Nested Directories) ---
# This calculates the path from HOME to the project, e.g., Documents/tg-llm-multibot
REL_PATH_FROM_HOME=$(echo "$REAL_PROJECT_ROOT" | sed "s|^$HOME/||")

HOME_SPECIFIER="%h"
VIRTUAL_ROOT="$HOME_SPECIFIER/$REL_PATH_FROM_HOME"
VIRTUAL_ENV_FILE="/etc/$PROJECT_DIR_NAME/.env"

# Decoupled System Paths
VIRTUAL_CONFIG_DIR="/etc/$PROJECT_DIR_NAME/confs"
VIRTUAL_HISTORY_DIR="$HOME_SPECIFIER/.local/share/$PROJECT_DIR_NAME/history"
ABS_HISTORY_DIR="$HOME/.local/share/$PROJECT_DIR_NAME/history"

# Persistent Data Path (Hidden in ~/.local/share)
VIRTUAL_PERSISTENT_DATA="$HOME_SPECIFIER/.local/share/ollama-data"
ABS_PERSISTENT_DATA="$HOME/.local/share/ollama-data"

STATUS_COL=35

# --- Color setup ---
if [ -t 1 ]; then
    if command -v tput >/dev/null 2>&1; then
        RED="$(tput setaf 1)"; GREEN="$(tput setaf 2)"; YELLOW="$(tput setaf 3)"
        BLUE="$(tput setaf 4)"; BOLD="$(tput bold)"; RESET="$(tput sgr0)"
    else
        RED='\033[0;31m'; GREEN='\033[0;32m'; YELLOW='\033[0;33m'
        BLUE='\033[0;34m'; BOLD='\033[1m'; RESET='\033[0m'
    fi
else
    RED=''; GREEN=''; YELLOW=''; BLUE=''; BOLD=''; RESET=''
fi

# --- Step function ---
step() {
    local msg="$1"; shift
    printf "%s" "$msg"
    local msg_len=$(printf "%s" "$msg" | wc -m)
    local spaces=$((STATUS_COL - msg_len))
    if [ $spaces -lt 1 ]; then spaces=1; fi
    printf "%${spaces}s"
    tmp_out=$(mktemp); tmp_err=$(mktemp)
    set +e
    "$@" >"$tmp_out" 2>"$tmp_err"
    exitcode=$?
    set -e
    if [ $exitcode -eq 0 ]; then
        echo "${GREEN}[ OK ]${RESET}"; rm -f "$tmp_out" "$tmp_err"
    else
        echo "${RED}[ ERR ]${RESET}"; echo
        echo "${RED}${BOLD}Step Failed:${RESET} $msg"
        echo "${BLUE}Command:${RESET}    $*"
        echo "${YELLOW}--- stdout ---${RESET}"; cat "$tmp_out"
        echo "${YELLOW}--- stderr ---${RESET}"; cat "$tmp_err"
        rm -f "$tmp_out" "$tmp_err"
        exit $exitcode
    fi
}

# --- Initialization ---
printf "${BOLD}Starting Quadlet Transformation${RESET}\n"

step "Checking Podlet" sh -c "command -v podlet >/dev/null 2>&1 || exit 1"
step "Checking Podman version" sh -c "podman version --format '{{.Client.Version}}' | grep -qE '^4\.[4-9]|^[5-9]\.'"

step "Creating empty output directory" sh -c "rm -rf \"$OUTPUT_DIR\" && mkdir -p \"$OUTPUT_DIR\""
step "Ensuring quadlet directory" mkdir -p "$QUADLET_DIR"

# --- Generation Logic ---
generate() {
    local file_name="$1"
    local base_name="${file_name%.*}" 
    local source_file="${INPUT_DIR}/$file_name"
    local target_subdir="$OUTPUT_DIR/$base_name"
    local temp_file="$target_subdir/temp_$file_name"

    if [ -f "$source_file" ]; then
        mkdir -p "$target_subdir"
        cp "$source_file" "$temp_file"

        # --- PRE-PROCESSING ---
        sed -i 's|file: .*api_keys.*|external: true|g' "$temp_file"
        sed -i -E '/^\s*(build|context|dockerfile):/d' "$temp_file"
        sed -i '/depends_on:/,/condition: service_healthy/d' "$temp_file"
        sed -i -E '/^\s*(deploy|resources|reservations|devices|capabilities):/d' "$temp_file"
        sed -i -E '/^\s*-\s*driver:\s*nvidia/d' "$temp_file"
        sed -i -E '/^\s*count:\s*all/d' "$temp_file"

        # --- GENERATION ---
        step "Generating $file_name" \
            podlet --file "$target_subdir" --overwrite compose "$temp_file"
        
        rm -f "$temp_file"

        # --- POST-PROCESSING ---
        step "  -> Injecting Env & GPU" sh -c "
            find \"$target_subdir\" -type f -name \"*.container\" | while read -r f; do
                # 1. Inject runtime .env loading (decoupling)
                sed -i \"/^\[Container\]$/a EnvironmentFile=$VIRTUAL_ENV_FILE\" \"\$f\"
                # 2. Inject GPU and Group Permissions
                if echo \"\$f\" | grep -q \"ollama\"; then
                    sed -i '/^\[Container\]$/a PodmanArgs=--device nvidia.com/gpu=all' \"\$f\"
                    sed -i '/^\[Container\]$/a Annotation=run.oci.keep_original_groups=1' \"\$f\"
                fi
                printf \"\n[Service]\nTimeoutStartSec=0\n\" >> \"\$f\"
            done
        "
    fi
}

generate "all.yml"
generate "pc.yml"
generate "pi.yml"

# --- Path Fixer (Global & Recursive) ---
step "Universalizing paths + Pre-creating volumes" sh -c "
    find \"$OUTPUT_DIR\" -type f \( -name \"*.container\" -o -name \"*.volume\" \) | while read -r f; do
        # 1. Aggressively catch ollama-data regardless of how podlet mangled the prefix
        sed -i \"s|Volume=[^ ]*ollama-data|Volume=$VIRTUAL_PERSISTENT_DATA|g\" \"\$f\"

        # 2. Fix confs and history specifically
        sed -i \"s|Volume=[^ ]*confs|Volume=$VIRTUAL_CONFIG_DIR|g\" \"\$f\"
        sed -i \"s|Volume=[^ ]*history|Volume=$VIRTUAL_HISTORY_DIR|g\" \"\$f\"

        # 3. Catch-all for any remaining project-root references
        sed -i \"s|$REAL_PROJECT_ROOT|$VIRTUAL_ROOT|g\" \"\$f\"
        
        # 4. Add :Z to volumes for SELinux safety
        sed -i '/^Volume=/ s/$/:Z/' \"\$f\"
        sed -i 's/:Z:Z/:Z/g' \"\$f\"
    done

    # 5. AUTO-CREATE missing host folders
    mkdir -p \"$ABS_PERSISTENT_DATA\"
    mkdir -p \"$ABS_HISTORY_DIR\"
"

# --- Final message ---
cat << EOF

${BOLD}==================================${RESET}
${GREEN}GENERATION COMPLETE!${RESET} -> ${BLUE}$OUTPUT_DIR_SHORT${RESET}

${BOLD}SETUP GUIDE:${RESET}
${BOLD}1. AUTO-BOOT${RESET}
${YELLOW}loginctl enable-linger \$USER${RESET}

${BOLD}2. SYSTEM CONFIGURATION (Requires sudo)${RESET}
${YELLOW}sudo mkdir -p /etc/$PROJECT_DIR_NAME${RESET}
${YELLOW}sudo cp .env $VIRTUAL_ENV_FILE${RESET}
${YELLOW}sudo cp -r confs/* /etc/$PROJECT_DIR_NAME/confs/ (if needed)${RESET}
${YELLOW}sudo chmod -R 644 /etc/$PROJECT_DIR_NAME && sudo find /etc/$PROJECT_DIR_NAME -type d -exec chmod 755 {} +${RESET}

${BOLD}3. DEPLOYMENT${RESET}
${YELLOW}rsync -av --delete $OUTPUT_DIR_NAME/pc/ $QUADLET_DIR_SHORT/${RESET}
${YELLOW}systemctl --user daemon-reload && systemctl --user restart ollama.service${RESET}

${BOLD}==================================${RESET}
EOF

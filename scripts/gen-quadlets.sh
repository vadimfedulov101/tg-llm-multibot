#!/bin/sh
set -e

# The script generates Quadlet configs from Docker Compose configs.
# Podman processes are booted from Quadlet configs as Systemd services.
#
# Properties:
# Agnostic: User (%h), Kernel (:Z)
# Decoupled: Config (/etc/project/.env)

# --- Configuration ---
# Assume the script is inside `project/scripts/`
PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
PROJECT_DIR_NAME=$(basename "$PROJECT_ROOT")

# EXTRA DIRECTORY NAMES (constants)
INPUT_DIR_NAME="docker-compose-ymls"
OUTPUT_DIR_NAME="podman-quadlets"
QUADLET_DIR_NAME=".config/containers/systemd"

# ABSOLUTE PATHS (for script execution)
INPUT_DIR="${PROJECT_ROOT}/${INPUT_DIR_NAME}"
OUTPUT_DIR="${PROJECT_ROOT}/${OUTPUT_DIR_NAME}"
QUADLET_DIR="${HOME}/${QUADLET_DIR_NAME}"

# SHORT PATHS (for logging)
OUTPUT_DIR_SHORT="${PROJECT_DIR_NAME}/${OUTPUT_DIR_NAME}"
QUADLET_DIR_SHORT="~/${QUADLET_DIR_NAME}"

# USER-AGNOSTIC PATHS (for portability)
HOME_SPECIFIER="%h"
VIRTUAL_ROOT="$HOME_SPECIFIER/$PROJECT_DIR_NAME"

# GLOBAL CONFIG PATH (The Idea: Store .env in /etc to decouple from user home)
VIRTUAL_ENV_FILE="/etc/$PROJECT_DIR_NAME/.env"

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
    local msg="$1"
    shift
    printf "%s" "$msg"
    local msg_len=$(printf "%s" "$msg" | wc -m)
    local spaces=$((STATUS_COL - msg_len))
    if [ $spaces -lt 1 ]; then spaces=1; fi
    printf "%${spaces}s"

    tmp_out=$(mktemp)
    tmp_err=$(mktemp)
    
    set +e
    "$@" >"$tmp_out" 2>"$tmp_err"
    exitcode=$?
    set -e

    if [ $exitcode -eq 0 ]; then
        echo "${GREEN}[ OK ]${RESET}"
        rm -f "$tmp_out" "$tmp_err"
    else
        echo "${RED}[ ERR ]${RESET}"
        echo
        echo "${RED}${BOLD}Step Failed:${RESET} $msg"
        echo "${BLUE}Command:${RESET}    $*"
        echo "${YELLOW}--- stdout ---${RESET}"
        cat "$tmp_out"
        echo "${YELLOW}--- stderr ---${RESET}"
        cat "$tmp_err"
        rm -f "$tmp_out" "$tmp_err"
        exit $exitcode
    fi
}

# --- Initialization ---
printf "${BOLD}🚀 Starting Quadlet Transformation${RESET}\n"

# --- Requirements Check ---
step "Checking Podlet" sh -c \
    "command -v podlet >/dev/null 2>&1 || \
    { echo >&2 'ERROR: podlet is not installed. Install via cargo.'; exit 1; }"
step "Checking Podman version" sh -c \
    "MIN_VER=4.4; CUR_VER=\$(podman version --format '{{.Client.Version}}'); \
    [ \"\$(printf '%s\\n%s' \"\$MIN_VER\" \"\$CUR_VER\" | sort -V | head -n1)\" = \"\$MIN_VER\" ]"

# --- Directory Management ---
step "Creating empty output directory" sh -c \
    "rm -rf \"$OUTPUT_DIR\" && mkdir -p \"$OUTPUT_DIR\""
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
        # Patch 1: Secrets (File -> External)
        sed -i 's|file: .*api_keys.*|external: true|g' "$temp_file"
        # Patch 2: Strip 'build'
        sed -i -E '/^\s*(build|context|dockerfile):/d' "$temp_file"
        # Patch 3: Strip 'depends_on'
        sed -i '/depends_on:/,/condition: service_healthy/d' "$temp_file"
        # Patch 4: Strip GPU specific markers
        sed -i -E '/^\s*(deploy|resources|reservations|devices|capabilities):/d' "$temp_file"
        # Patch 5: Strip GPU 'driver'
        sed -i -E '/^\s*-\s*driver:\s*nvidia/d' "$temp_file"
        # Patch 6: Strip GPU 'count'
        sed -i -E '/^\s*count:\s*all/d' "$temp_file"

        # --- GENERATION ---
        step "Generating $file_name" \
            podlet --file "$target_subdir" --overwrite compose "$temp_file"
        
        rm -f "$temp_file"

        # --- POST-PROCESSING ---
        # Injected Decoupling Logic: Link runtime .env and inject GPU
        step "  -> Injecting Env & GPU" sh -c "
            find \"$target_subdir\" -type f -name \"*.container\" | while read -r f; do
                # Inject runtime .env loading (Decoupling)
                # Pointed to the global /etc path instead of user home
                sed -i \"/^\[Container\]$/a EnvironmentFile=$VIRTUAL_ENV_FILE\" \"\$f\"
                
                # Check if this container needs GPU (Nvidia)
                if echo \"\$f\" | grep -q \"ollama\"; then
                    sed -i '/^\[Container\]$/a PodmanArgs=--device nvidia.com/gpu=all' \"\$f\"
                fi
            done
        "
    else
        echo "${YELLOW}Warning: $file_name not found, skipping.${RESET}"
    fi
}

# Process files
generate "all.yml"
generate "pc.yml"
generate "pi.yml"

# --- Path Fixer (Global & Recursive) ---
step "Universalizing paths + kernel" sh -c "
    find \"$OUTPUT_DIR\" -type f \( -name \"*.container\" -o -name \"*.volume\" \) | while read -r f; do
        # 1. Replace real path with %h specifier
        sed -i \"s|$PROJECT_ROOT|$VIRTUAL_ROOT|g\" \"\$f\"
        # 2. Add :Z to volumes for SELinux safety (if not already present)
        sed -i '/^Volume=/ s/$/:Z/' \"\$f\"  # Append :Z to path
        sed -i 's/:Z:Z/:Z/g' \"\$f\"  # Clean up possible double :Z
    done
"

# --- Final message ---
cat << EOF

${BOLD}==================================${RESET}
${GREEN}GENERATION COMPLETE!${RESET} -> ${BLUE}$OUTPUT_DIR_SHORT${RESET}

${BOLD}SETUP GUIDE:${RESET}
${BOLD}0. CHANGE DIRECTORY TO ROOT${RESET}
${YELLOW}cd ${PROJECT_ROOT}${RESET}

${BOLD}1. AUTO-BOOT${RESET}
${YELLOW}loginctl enable-linger $USER${RESET}

${BOLD}2. CONFIG${RESET}
${YELLOW}sudo mkdir -p /etc/$PROJECT_DIR_NAME${RESET}
${YELLOW}sudo cp .env $VIRTUAL_ENV_FILE${RESET}

${BOLD}3. DEPLOYMENT${RESET}
${BLUE}# Select pc/ollama or pi/tg-handler consistently${RESET}
${YELLOW}cp -ru $OUTPUT_DIR_SHORT/(pc|pi)/ $QUADLET_DIR_SHORT/${RESET}
${YELLOW}systemctl --user daemon-reload${RESET}
${YELLOW}systemctl --user restart (ollama|tg-handler).service${RESET}
${BLUE}# For pi/tg-handler also set API keys${RESET}
${YELLOW}podman secret create api_keys ./api_keys.txt${RESET}

EOF

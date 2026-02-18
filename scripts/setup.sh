#!/bin/sh
set -e

# --- Configuration ---
PROJECT_ETC="/etc/tg-llm-multibot"
PROJECT_DATA="$HOME/.local/share"
LLM_DIR="$PROJECT_DATA/ollama-data"
HIST_DIR="$PROJECT_DATA/tg-data/history"
QUADLET_DIR="$HOME/.config/containers/systemd"

# Helper for source location of quadlet files
INPUT_QUADLETS="podman-quadlets"

STATUS_COL=35

# --- Color & Formatting setup ---
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

# Cursor manipulation for the "Clean Sudo" effect
UP_LINE='\033[1A'
CLEAR_LINE='\033[2K'
WIPE_LINE="${UP_LINE}${CLEAR_LINE}"

# --- CLI Parsing ---
usage() {
    echo "Usage: $0 [-d | -p]"
    echo "  -d   Setup for Docker Compose"
    echo "  -p   Setup for Podman Quadlets"
    exit 1
}

MODE=""
while getopts "dp" opt; do
    case $opt in
        d) MODE="docker" ;;
        p) MODE="podman" ;;
        *) usage ;;
    esac
done

if [ -z "$MODE" ]; then
    printf "${RED}Error: You must specify a mode.${RESET}\n"
    usage
fi


# --- Robust GPU Check ---
# Returns 0 if NVIDIA GPU is detected, 1 otherwise
check_gpu() {
    # 1. Try lspci (Look for Vendor ID 10de = NVIDIA)
    if command -v lspci >/dev/null 2>&1; then
        if lspci -d 10de: -nn | grep -q .; then
            return 0
        fi
    fi

    # 2. Try sysfs (Linux kernel device tree) - works even if lspci is missing
    # grep returns 0 if match found
    if grep -qi "0x10de" /sys/bus/pci/devices/*/vendor 2>/dev/null; then
        return 0
    fi

    # 3. Fallback: Try nvidia-smi (implies driver is loaded)
    if command -v nvidia-smi >/dev/null 2>&1; then
        return 0
    fi

    return 1
}

# --- Step function ---
step() {
    local msg="$1"; shift
    printf "%s" "$msg"
    local msg_len=$(printf "%s" "$msg" | wc -m)
    local spaces=$((STATUS_COL - msg_len))
    if [ $spaces -lt 1 ]; then spaces=1; fi
    printf "%${spaces}s"
    tmp_out=$(mktemp); tmp_err=$(mktemp)
    
    # Allow CMD to fail inside the function for printing the error
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
        # Exit immediately on fail
        exit $exitcode
    fi
}

# --- Initialization ---
printf "${BOLD}Initializing Setup for ${BLUE}${MODE^^}${RESET}\n"

# --- Sudo Check ---
# Check if sudo token cached (-n = non-interactive)
if sudo -n true 2>/dev/null; then
    # Token present, refresh the timer
    sudo -v
else
    # Token expired or missing, refresh the credentials
    printf "${BOLD}Authorization required for setup:${RESET}\n"
    if ! sudo -v; then
        step "Sudo Access" false
    fi
    
    # Success: wipe header + password prompt
    printf "%b%b" "$WIPE_LINE" "$WIPE_LINE"
fi
step "Sudo Access" true

# --- Dependency Checks ---
if [ "$MODE" = "podman" ]; then
    # Podman: 4.4+ or 5+
    step "Checking Podman binary" sh -c "command -v podman >/dev/null 2>&1"
    step "Checking Podman version" sh -c \
        "podman version --format '{{.Client.Version}}' | grep -qE '^4\.[4-9]|^[5-9]\.'"
    step "Enabling User Linger" loginctl enable-linger "$USER"

elif [ "$MODE" = "docker" ]; then
    step "Checking Docker binary" sh -c "command -v docker >/dev/null 2>&1"
    step "Checking Docker Compose" sh -c "docker compose version >/dev/null 2>&1"
fi

# --- 3. Common System Preparation ---
step "Creating Dirs" sudo mkdir -p "$PROJECT_ETC" "$LLM_DIR" "$HIST_DIR"
step "Copying Files" sh -c "
    sudo cp api_keys.txt \"$PROJECT_ETC/\"
    sudo cp ollama.env \"$PROJECT_ETC/\"
    sudo cp -r confs \"$PROJECT_ETC/\"
    sudo cp scripts/ollama-entrypoint.sh \"$PROJECT_ETC/\"
"

# --- 4. Permission Setting ---
set_etc_rights() {
    local dir="$1"
    # Etc: Root-owned, but Group-readable
    sudo chown -R root:"$USER" "$dir"
    sudo chmod -R 640 "$dir"
    sudo find "$dir" -type d -exec chmod 750 {} +
}

set_data_rights() {
    local dir="$1"
    # Data: User-owned (rootless)
    sudo chown -R "$USER":"$USER" "$dir"
    chmod -R 755 "$dir"
}

step "Securing $PROJECT_ETC ownership" set_etc_rights "$PROJECT_ETC"
step "Securing $LLM_DIR ownership" set_data_rights "$LLM_DIR"
step "Securing $HIST_DIR ownership" set_data_rights "$HIST_DIR"
step "Making entrypoint executable" sudo chmod +x "$PROJECT_ETC/ollama-entrypoint.sh"

# --- Mode-Specific Deployment ---
if [ "$MODE" = "podman" ]; then
    printf "\n${BOLD}Deployment: Podman Quadlet${RESET}\n"
    
    step "Creating Quadlet Dir" mkdir -p "$QUADLET_DIR"

    # Deduce container to set
    TARGET_CONTAINER=""
    if check_gpu; then
        # GPU: Deploy Ollama (PC logic)
        TARGET_CONTAINER="ollama.container"
        printf "  -> ${GREEN}GPU Detected.${RESET} Deploying: ${BOLD}$TARGET_CONTAINER${RESET}\n"
    else
        # No GPU: Deploy TG-Handler (Pi logic)
        TARGET_CONTAINER="tg-handler.container"
        printf "  -> ${YELLOW}No GPU Detected.${RESET} Deploying: ${BOLD}$TARGET_CONTAINER${RESET}\n"
    fi

    # Check conatiner presence
    if [ ! -f "$INPUT_QUADLETS/$TARGET_CONTAINER" ]; then
        echo "${RED}Error: Source file '$INPUT_QUADLETS/$TARGET_CONTAINER' not found.${RESET}"
        exit 1
    fi
    
    # Deploy and reload SystemD
    step "Deploying $TARGET_CONTAINER" sh -c "
        cp \"$INPUT_QUADLETS/$TARGET_CONTAINER\" \"$QUADLET_DIR/\"
    "
    step "Reloading SystemD" systemctl --user daemon-reload
    
    # Deduce service name from container name (remove extension)
    SERVICE_NAME="${TARGET_CONTAINER%.*}.service"

    cat << EOF

${BOLD}==================================${RESET}
${GREEN}PODMAN SETUP COMPLETE!${RESET}

Service deployed: ${BOLD}$SERVICE_NAME${RESET}
Start it with:
${YELLOW}systemctl --user enable --now $SERVICE_NAME${RESET}
EOF

elif [ "$MODE" = "docker" ]; then
    printf "\n${BOLD}Deployment: Docker Compose${RESET}\n"
    
    cat << EOF

${BOLD}==================================${RESET}
${GREEN}DOCKER SETUP COMPLETE!${RESET}

Start services:
${YELLOW}docker compose up -f docker-compose-ymls/pc.yml -d${RESET}
${YELLOW}docker compose up -f docker-compose-ymls/pi.yml -d${RESET}
EOF

fi

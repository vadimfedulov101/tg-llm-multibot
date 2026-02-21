#!/bin/sh
set -e

# --- Configuration ---
# Directories to create
DATA_DIR="$HOME/.local/share"
CONF_DIR="$HOME/.config"

TELELLAMA_DIR="$CONF_DIR/telellama"
OLLAMA_DIR="$DATA_DIR/ollama-data"
BOTHIST_DIR="$DATA_DIR/bots-data/history"

# Files/directories to copy
SECRET_FILE="api_keys.txt"
ENV_FILE="ollama.env"
BOTCONF_DIR="confs"
ENTRYPOINT_DIR="scripts"
ENTRYPOINT_FILENAME="ollama-entrypoint.sh"
ENTRYPOINT_FILE="$ENTRYPOINT_DIR/$ENTRYPOINT_FILENAME"

# Containers
DOCKER_DIR="containers/docker"
PODMAN_DIR="containers/podman"
OLLAMA_PODMAN_CONTAINER="$PODMAN_DIR/ollama.container"
BOTS_PODMAN_CONTAINER="$PODMAN_DIR/bots.container"
OLLAMA_DOCKER_CONTAINER="$DOCKER_DIR/ollama.yml"
BOTS_DOCKER_CONTAINER="$DOCKER_DIR/bots.yml"

# Directory for Podman containers
QUADLET_DIR="$CONF_DIR/containers/systemd"


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
printf "${BOLD}Initializing Setup for ${BLUE}$(echo "$MODE" | tr '[:lower:]' '[:upper:]')${RESET}\n"

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
step "Creating Dirs" mkdir -p "$TELELLAMA_DIR" "$OLLAMA_DIR" "$BOTHIST_DIR"
step "Copying Files" sh -c "
    cp \"$SECRET_FILE\" \"$TELELLAMA_DIR\"
    cp \"$ENV_FILE\" \"$TELELLAMA_DIR\"
    cp -r \"$BOTCONF_DIR\" \"$TELELLAMA_DIR\"
    cp \"$ENTRYPOINT_FILE\" \"$TELELLAMA_DIR\"
"

# --- 4. Permission Setting ---
set_perms() {
    local dir="$1"

    sudo chown -R "$USER":"$USER" "$dir"
    sudo chmod -R 640 "$dir"
    sudo find "$dir" -type d -exec chmod 750 {} +
}

apply_all_perms() {
    set_perms "$TELELLAMA_DIR"
    set_perms "$OLLAMA_DIR"
    set_perms "$BOTHIST_DIR"
    # Handle the executable
    local entrypoint_dst_file="$TELELLAMA_DIR/$ENTRYPOINT_FILENAME"
    chmod +x "$entrypoint_dst_file"
}

step "Setting Dirs & Files permissions" apply_all_perms

# --- Mode-Specific Deployment ---
if [ "$MODE" = "podman" ]; then
    printf "\n${BOLD}Deployment: Podman Quadlet${RESET}\n"
    
    step "Creating Quadlet Dir" mkdir -p "$QUADLET_DIR"

    # Deduce container to deploy
    TARGET_CONTAINER=""
    if check_gpu; then
        # GPU: Deploy Ollama for PC
        TARGET_CONTAINER="$OLLAMA_PODMAN_CONTAINER"
        printf "  -> ${GREEN}GPU Detected.${RESET}\n"
    else
        # No GPU: Deploy bots for Pi
        TARGET_CONTAINER="$BOTS_PODMAN_CONTAINER"
        printf "  -> ${YELLOW}No GPU Detected.${RESET}\n"
    fi

    # Deploy and reload SystemD
    step "Deploying $(basename $TARGET_CONTAINER)" sh -c "
        cp \"$TARGET_CONTAINER\" \"$QUADLET_DIR/\"
    "
    step "Reloading SystemD" systemctl --user daemon-reload
    
    # Deduce target service from target container
    # 1. Get filename
    FILENAME=$(basename "$TARGET_CONTAINER")
    # 2. Remove extension, add .service
    TARGET_SERVICE="${FILENAME%.*}.service"
    
    cat << EOF

${BOLD}==================================${RESET}
${GREEN}PODMAN SETUP COMPLETE!${RESET}

Service deployed: ${BOLD}$TARGET_SERVICE${RESET}

Start it with:
${YELLOW}systemctl --user start $TARGET_SERVICE${RESET}
EOF

elif [ "$MODE" = "docker" ]; then
    printf "\n${BOLD}Deployment: Docker Compose${RESET}\n"

    # Deduce container to deploy
    TARGET_CONTAINER=""
    if check_gpu; then
        # GPU: Deploy Ollama for PC
        TARGET_CONTAINER="$OLLAMA_DOCKER_CONTAINER"
        printf "  -> ${GREEN}GPU Detected.${RESET}"
    else
        # No GPU: Deploy bots for Pi
        TARGET_CONTAINER="$BOTS_DOCKER_CONTAINER"
        printf "  -> ${YELLOW}No GPU Detected.${RESET}"
    fi
    printf " Deploying: ${BOLD}$TARGET_CONTAINER${RESET}\n"

    # Deduce target YML from target container
    # 1. Get filename
    FILENAME=$(basename "$TARGET_CONTAINER")
    # 2. Remove extension, add .yml
    TARGET_YML="${FILENAME%.*}.yml"

    cat << EOF

${BOLD}==================================${RESET}
${GREEN}DOCKER SETUP COMPLETE!${RESET}

Container to deploy: ${BOLD}$TARGET_YML${RESET}

${BOLD}IMPORTANT: Permission Setup${RESET}
To allow Docker to correctly find your configuration in ${BLUE}\$HOME${RESET}, 
you should run Docker without sudo.

1. Add your user to the docker group (if not already done):
   ${YELLOW}sudo usermod -aG docker \$USER${RESET}

2. Apply the group change to your current session:
   ${YELLOW}newgrp docker${RESET}
   ${BOLD}OR${RESET} log out and log back in.

3. Start the container:
   ${YELLOW}docker compose -f containers/docker/$TARGET_YML up -d${RESET}
EOF

fi

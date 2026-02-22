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
OLLAMA_DOCKER_CONTAINER="$DOCKER_DIR/ollama.yml"
BOTS_DOCKER_CONTAINER="$DOCKER_DIR/bots.yml"
OLLAMA_PODMAN_CONTAINER="$PODMAN_DIR/ollama.container"
BOTS_PODMAN_CONTAINER="$PODMAN_DIR/bots.container"
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
    step "Enabling User Linger" sudo loginctl enable-linger "$USER"

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

# Sets option based on GPU presence
set_based_on_gpu() {
    local gpu_file=$1
    local non_gpu_file=$2
    local selected=""

    if check_gpu; then
        printf "  -> ${GREEN}GPU Detected.${RESET}\n" >&2
        selected="$gpu_file"
    else 
        printf "  -> ${YELLOW}No GPU Detected.${RESET}\n" >&2
        selected="$non_gpu_file"
    fi

    # Print human-readable feedback via stderr
    printf "  File to deploy: ${BOLD}$(basename "$selected")${RESET}\n" >&2

    # Return via stdout
    echo "$selected"
}

setup_docker() {
    printf "\n${BOLD}Deployment: Docker Compose${RESET}\n"

    # Get container and YML
    TARGET_CONTAINER=$(set_based_on_gpu \
        "$OLLAMA_DOCKER_CONTAINER" "$BOTS_DOCKER_CONTAINER")
    TARGET_YML=$(basename "$TARGET_CONTAINER")

    # Elevate user rights
    step "Adding $USER to Docker group" sudo usermod -aG docker $USER

    cat << EOF
${BOLD}==================================${RESET}
${GREEN}DOCKER SETUP COMPLETE!${RESET}

${BOLD}Note:${RESET} Log out and back in for group changes to take effect.

Start the container (without sudo):
${YELLOW}docker compose -f containers/docker/$TARGET_YML up -d${RESET}
EOF
}

setup_podman() {
    printf "\n${BOLD}Deployment: Podman Quadlet${RESET}\n"
    
    step "Creating Quadlet Dir" mkdir -p "$QUADLET_DIR"

    # Get target container and service
    TARGET_CONTAINER=$(set_based_on_gpu \
        "$OLLAMA_PODMAN_CONTAINER" "$BOTS_PODMAN_CONTAINER")
    FILENAME=$(basename "$TARGET_CONTAINER")
    TARGET_SERVICE="${FILENAME%.*}.service"

    # Deploy container with SystemD reload
    step "Deploying $TARGET_SERVICE" cp "$TARGET_CONTAINER" "$QUADLET_DIR"
    step "Reloading SystemD" systemctl --user daemon-reload

    cat << EOF
${BOLD}==================================${RESET}
${GREEN}PODMAN SETUP COMPLETE!${RESET}

${BOLD}Note:${RESET} You may need to prebuild local container.
${YELLOW}podman build -t telellama-bots:local -f bots/Dockerfile .${RESET}

Start the service (without sudo):
${YELLOW}systemctl --user start $TARGET_SERVICE${RESET}
EOF
}

# --- Mode-Specific Deployment ---
if [ "$MODE" = "podman" ]; then
    setup_podman
elif [ "$MODE" = "docker" ]; then
    setup_docker
fi

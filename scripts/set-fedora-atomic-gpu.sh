#!/bin/sh
set -e

STATUS_COL=45

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
printf "${BOLD}Initializing Fedora Atomic GPU Setup${RESET}\n"

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

# --- Logic ---

# Check if NVIDIA Container Toolkit is already layered
if ! rpm-ostree status | grep -q "nvidia-container-toolkit"; then
    printf "\n${BOLD}[Phase 1/2] Layering NVIDIA Container Toolkit${RESET}\n"

    step "Adding NVIDIA Repository" sh -c "curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo >/dev/null"
    
    step "Layering NVIDIA Toolkit" sudo rpm-ostree install -y nvidia-container-toolkit
    
    step "Blacklisting Nouveau & Enabling DRM" sudo rpm-ostree kargs \
        --append="rd.driver.blacklist=nouveau" \
        --append="modprobe.blacklist=nouveau" \
        --append="nvidia-drm.modeset=1"
    
    step "Reconstructing Boot Image" sudo rpm-ostree initramfs --enable
    
    cat << EOF

${BOLD}==================================${RESET}
${GREEN}PHASE 1 COMPLETE!${RESET}

${YELLOW}${BOLD}ACTION REQUIRED:${RESET} You must reboot your system now to apply the ostree changes.
After rebooting, run this script one more time to generate the CDI config.
EOF
    exit 0
fi

printf "\n${BOLD}[Phase 2/2] Generating CDI Configuration${RESET}\n"

step "Creating /etc/cdi directory" sudo mkdir -p /etc/cdi

step "Generating NVIDIA CDI config" sudo nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml

step "Verifying CDI configuration" grep -q "nvidia.com/gpu" /etc/cdi/nvidia.yaml

cat << EOF

${BOLD}==================================${RESET}
${GREEN}GPU SETUP COMPLETE!${RESET}

Your Podman containers can now natively access the GPU using CDI.
Ensure your Quadlet (.container) files include:
${YELLOW}AddDevice=nvidia.com/gpu=all${RESET}
EOF

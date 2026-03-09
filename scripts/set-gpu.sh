#!/bin/sh
set -e

STATUS_COL=55

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

# --- Distribution Detection ---
if [ -f /etc/os-release ]; then
    . /etc/os-release
    OS_ID=$ID
else
    printf "${RED}Error: Cannot detect OS (${BOLD}/etc/os-release${RESET} missing).\n"
    exit 1
fi

IS_ATOMIC=0
if [ "$OS_ID" = "fedora" ]; then
    # Distinguish standard Fedora from Fedora Atomic (Silverblue/Kinoite/CoreOS)
    if command -v rpm-ostree >/dev/null 2>&1 &&[ -d /run/ostree-booted ]; then
        IS_ATOMIC=1
    fi
fi

if [ "$OS_ID" != "ubuntu" ] &&[ "$OS_ID" != "fedora" ]; then
    printf "${RED}Unsupported OS: ${BOLD}$OS_ID${RESET}\n"
    exit 1
fi

# --- Initialization ---
printf "${BOLD}Initializing GPU Setup for "
if [ "$OS_ID" = "ubuntu" ]; then
    printf "Ubuntu"
elif [ "$OS_ID" = "fedora" ] && [ $IS_ATOMIC -eq 0 ]; then
    printf "Fedora (Standard)"
else
    printf "Fedora Atomic"
fi
printf "${RESET}\n\n"

# --- Sudo Check ---
if sudo -n true 2>/dev/null; then
    sudo -v
else
    printf "${BOLD}Authorization required for setup:${RESET}\n"
    if ! sudo -v; then
        step "Sudo Access" false
    fi
    printf "%b%b" "$WIPE_LINE" "$WIPE_LINE"
fi
step "Sudo Access" true

# --- Toolkit Installation Check ---
TOOLKIT_INSTALLED=0
if [ "$OS_ID" = "ubuntu" ]; then
    if dpkg -s nvidia-container-toolkit >/dev/null 2>&1; then TOOLKIT_INSTALLED=1; fi
elif [ "$OS_ID" = "fedora" ]; then
    if [ $IS_ATOMIC -eq 1 ]; then
        if rpm-ostree status | grep -q "nvidia-container-toolkit"; then TOOLKIT_INSTALLED=1; fi
    else
        if rpm -q nvidia-container-toolkit >/dev/null 2>&1; then TOOLKIT_INSTALLED=1; fi
    fi
fi

# --- Phase 1: Installation Logic ---
if [ $TOOLKIT_INSTALLED -eq 0 ]; then
    printf "\n${BOLD}[Phase 1] Installing NVIDIA Container Toolkit${RESET}\n"

    if [ "$OS_ID" = "ubuntu" ]; then
        step "Adding NVIDIA GPG Key" sh -c "curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg --yes"
        step "Adding NVIDIA Repository" sh -c "curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | sed 's#deb https://#deb[signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list >/dev/null"
        step "Updating APT cache" sudo apt update -q
        step "Installing Toolkit" sudo apt install -y nvidia-container-toolkit

    elif [ "$OS_ID" = "fedora" ]; then
        step "Adding NVIDIA Repository" sh -c "curl -s -L https://nvidia.github.io/libnvidia-container/stable/rpm/nvidia-container-toolkit.repo | sudo tee /etc/yum.repos.d/nvidia-container-toolkit.repo >/dev/null"
        
        if [ $IS_ATOMIC -eq 1 ]; then
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
        else
            step "Installing Toolkit" sudo dnf install -y nvidia-container-toolkit
        fi
    fi
else
    printf "\n${BOLD}[Phase 1] NVIDIA Container Toolkit already installed. Skipping...${RESET}\n"
fi

# --- Phase 2: Generating CDI ---
printf "\n${BOLD}[Phase 2] Generating CDI Configuration${RESET}\n"

step "Creating /etc/cdi directory" sudo mkdir -p /etc/cdi

step "Generating NVIDIA CDI config" sudo nvidia-ctk cdi generate --output=/etc/cdi/nvidia.yaml

step "Verifying CDI configuration" grep -q "nvidia.com/gpu" /etc/cdi/nvidia.yaml

cat << EOF

${BOLD}==================================${RESET}
${GREEN}GPU SETUP COMPLETE!${RESET}

Your Podman/Docker containers can now natively access the GPU using CDI.
Ensure your configuration / Quadlet (.container) files include:
${YELLOW}AddDevice=nvidia.com/gpu=all${RESET}  (For Podman)
EOF

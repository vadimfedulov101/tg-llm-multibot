func install_cuda_toolkit () {
    url="https://developer.download.nvidia.com/compute/cuda/repos"
    distro="wsl-ubuntu"
    arch="x86_64"
    file="cuda-keyring_1.1-1_all.deb"

    # Get and install keyring file
    wget "$url/$distro/$arch/$file" && sudo dpkg -i "$file"

    # Install cuda toolkit metapackage (without driver)
    sudo apt update && sudo apt install -y --no-install-recommends \
        cuda-toolkit-13-0

    # Clean keyring file
    rm -f "$file"
}

func install_container_toolkit () {
    # Install dependencies
    sudo apt update && sudo apt install -y --no-install-recommends \
       curl \
       gnupg2

    # Get and install keyring
    curl -fsSL https://nvidia.github.io/libnvidia-container/gpgkey | sudo gpg --dearmor -o /usr/share/keyrings/nvidia-container-toolkit-keyring.gpg \
      && curl -s -L https://nvidia.github.io/libnvidia-container/stable/deb/nvidia-container-toolkit.list | \
        sed 's#deb https://#deb [signed-by=/usr/share/keyrings/nvidia-container-toolkit-keyring.gpg] https://#g' | \
        sudo tee /etc/apt/sources.list.d/nvidia-container-toolkit.list

    # Install container toolkit packages
    export NVIDIA_CONTAINER_TOOLKIT_VERSION=1.18.0-1
      sudo apt-get install -y \
          nvidia-container-toolkit=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
          nvidia-container-toolkit-base=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
          libnvidia-container-tools=${NVIDIA_CONTAINER_TOOLKIT_VERSION} \
          libnvidia-container1=${NVIDIA_CONTAINER_TOOLKIT_VERSION}
}

install_cuda_toolkit
install_container_toolkit

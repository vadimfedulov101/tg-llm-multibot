# Telellama 🦙💬

**Telellama** is a resilient, distributed, multi-bot Telegram framework powered by Ollama LLMs. 

## 🏗️ Architecture

**Distributed Deployment**:
* **Pi**: always-on, runs the bots.
* **PC**: on-demand, performs the inference.

**Error-free Flow**:
1. The bots (Pi) await new messages.
2. When triggered, they try to reach for the Ollama (PC).
3. If Ollama is unreachable, the bots eternally retry the generation request.

```mermaid
sequenceDiagram
    participant U as Telegram User
    box Pi (Always-On)
        participant B as Bots
        participant Q as Protobuf History
    end
    box PC (On-Demand)
        participant O as Ollama Server
    end

    U->>B: Sends Message
    B->>Q: Saves Message
    B->>B: Formats Prompts
    loop Eternal Retry (Every 10s)
        B->>O: HTTP POST /api/generate
        note right of B: If PC is OFF, bot waits<br/>not dropping the message.
    end
    O-->>B: Raw Response
    B->>B: Filters Response
    B->>Q: Saves Reply
    B->>U: Sends Reply
    B->>O: Evaluates User Tags
```

## 🚀 Quick Start

### 1. Pi Setup

1. Download ISO image (as `.xz`) for your [Pi](http://www.orangepi.org/html/hardWare/computerAndMicrocontrollers/details/Orange-Pi-Zero-2W.html) from the [DietPi website](https://dietpi.com/#download) 
2. Burn the image into [SD-card](https://www.sandisk.com/en-se/products/memory-cards/microsd-cards/sandisk-ultra-lite-uhs-i-microsd?sku=SDSQUNR-032G-GN3MA) with [Rufus](https://rufus.ie/en/) (`.xz` supported) or other program.
3. Set your variables in `set-dietpi.sh` and run it on the burned SD-card.
    ```
   ./set-dietpi.sh
   ```
4. Start up your Pi and SSH into it.
    ```
    ssh root:192.168.0.102
    ```
5. Set up your Pi.
    ```
    # Install the requirements
    sudo apt update
    sudo apt install -y podman dbus-user-session uidmap passt slirp4netns

    # Create the user with home and bash
    sudo useradd -m -s /bin/bash tgbot
    sudo adduser tgbot sudo

    # Unmask logind
    systemctl unmask systemd-logind

    # Start the services
    systemctl start systemd-logind
    systemctl start dbus

    # Enter the user correctly
    sudo -i -u tgbot
    ```

### 2. PC Setup

1. `git clone https://github/vadimfedulov101/telellama`
2. `cd telellama`
3. Set static IP address.
    ```
    # 1. Set the IP address
    sudo nmcli connection modify "enp3s0" ipv4.addresses "192.168.0.101/24"

    # 2. Set the Gateway
    sudo nmcli connection modify "enp3s0" ipv4.gateway `ip route show default | awk '{print $3}'`

    # 3. Set DNS
    sudo nmcli connection modify "enp3s0" ipv4.dns "8.8.8.8,1.1.1.1"

    # 4. Set to Manual
    sudo nmcli connection modify "enp3s0" ipv4.method manual

    # 5. Apply
    sudo nmcli connection up "enp3s0"
    ```
*Note: Your router DHCP-range must exclude just set IP address (e.g. 192.168.0.101). Settings are located via Gateway IP `ip route show default | awk '{print $3}` as a link.*

4. Configure `ollama.env` file to point to just set static IP.
    ```env
    OLLAMA_API_URL=http://192.168.1.101:11434/api/generate
    ```

5. Allow the Ollama port.
    ```bash
    # 1. Allow port 11434 through the firewall
    sudo firewall-cmd --add-port=11434/tcp --permanent
    # 2. Reload to apply changes
    sudo firewall-cmd --reload
    ```
WSL: `scripts/set-wsl-ports.ps1`

### 3. Container Setup (Docker / Podman Agnostic)

**Engine-agnostic**:
* Docker Compose: `./scripts/setup-containers.sh -d`
* Podman Quadlets: `./scripts/setup-containers.sh -p`

The container to set is **auto-deduced** by GPU presence.

## ⚙️ Configuration

Configurations are loaded on the boot from JSON files:

*   **Global Settings (`./confs/init.json`):**

    Defines allowed chat IDs, prompt templates, memory limits, message time-to-live (TTL), and default LLM parameters (Temperature, Top K, etc.).

*   **Bot-Specific Settings (`./confs/bots/<botname>.json`):**

    Defines the persona prompt and the number of response candidates to generate.

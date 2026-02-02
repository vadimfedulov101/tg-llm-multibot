# TG-LLM-MultiBot 🚀

Advanced Framework for Multi-Bot LLM Orchestration in Telegram using Go, Ollama, and Protobuf.

## ✨ Key Features
*   **Multi-Persona Management**: Run distinct characters (e.g., *Revy*, *Frieren*) with unique system prompts and memories concurrently.
*   **Ollama Integration**: Self-hosted LLM inference using the `ollama` API.
*   **Intelligent Pipeline**:
    *   **Candidate Generation**: Generates multiple response options per turn.
    *   **Selection**: Uses the LLM to pick the most authentic response based on character constraints.
    *   **Reflection**: Auto-generates "Carma" (Karma) updates and memory tags based on user interaction.
    *   **Denoising & Translation**: Cleans `<think>` blocks and auto-translates outputs (e.g., to Esperanto/English) if configured.
*   **Protobuf Memory**: Efficient, binary-serialized conversation history (`history.pb`) split into shared queues (public chats) and private chains.
*   **Distributed Architecture**: Deploy components on a single machine or split between a low-power edge device (RPi/DietPi) and a high-power GPU server.

## 🏗 Architecture

### 📊 Component Flow

```mermaid
graph TD
    User[👤 Telegram User] <-->|Messages| TG[🐹 TG Handler<br/>(Go Container)]
    
    subgraph Edge_or_Server [Docker: TG Handler]
        TG -->|Load/Save| Proto[💾 history.pb<br/>(Protobuf)]
        TG -->|Read Config| Json[Cx JSON Configs]
    end

    subgraph GPU_Server [Docker: Ollama]
        Ollama[🦙 Ollama API]
    end

    TG -->|1. Generate Candidates| Ollama
    TG -->|2. Select Best Candidate| Ollama
    TG -->|3. Update Tags/Carma| Ollama
    
    Ollama -->|JSON Responses| TG
    
    %% Styling
    style User fill:#E3F2FD,stroke:#0088CC
    style TG fill:#E0F7FA,stroke:#00ADD8,stroke-width:2px
    style Ollama fill:#FFF3E0,stroke:#FFB300,stroke-width:2px
    style Proto fill:#F3E5F5,stroke:#AB47BC
    style Json fill:#F3E5F5,stroke:#AB47BC
```

### Core Components

| Component | Technology | Description |
| :--- | :--- | :--- |
| **TG Handler** | Go 1.25+ | Orchestrates the entire logic. Handles Telegram updates, manages Protobuf memory (queues/chains), cleans inputs, and decides when to call the LLM. |
| **Ollama** | Docker/C++ | Provides the inference API. Supports models like `qwen2.5`, `mistral`, etc. Handles the heavy lifting of text generation. |
| **Memory** | Protobuf | `history.pb`. Stores chat queues (shared for public chats) and reply chains. Includes "Carma" and "Tags" per user. |
| **Cleaner** | Go Routine | Background process that deletes messages older than `msg_ttl` (default ~186h) to keep context relevant. |

## 🛠️ Setup & Scripts

The `scripts/` directory contains utilities to set up your environment.

### 1. 🐧 Linux / DietPi Setup
If you are running the **TG Handler** on a Raspberry Pi or DietPi:

1.  **Network Setup**: Edit `scripts/set-dietpi.sh` with your Gateway, IP, and WiFi credentials, then run it to configure static IP and WiFi.
    ```bash
    sudo ./scripts/set-dietpi.sh
    ```
2.  **Docker Setup**: Installs Docker engine.
    ```bash
    sudo ./scripts/set-docker.sh
    ```

### 2. 🖥️ GPU Server (NVIDIA) Setup
If you are hosting **Ollama** on a Linux machine with an NVIDIA GPU:

1.  **NVIDIA Container Toolkit**: Installs drivers and toolkit to allow Docker to access the GPU.
    ```bash
    sudo ./scripts/set-nvidia.sh
    ```

### 3. 🪟 Windows (WSL2) Setup
If you are running **Ollama** inside WSL2 on Windows, you need to forward port `11434` from WSL to the Windows host so the TG Handler (on another device) can reach it.

**How to automate `scripts/set-wsl-ports.ps1`:**
*Issue:* Simply running the script keeps a PowerShell window open or asks for Admin rights every boot.
*Fix:* Use Windows Task Scheduler.

1.  Open **Task Scheduler** (Run `taskschd.msc`).
2.  Click **Create Task**.
3.  **General Tab**:
    *   Name: `WSL Port Forward`
    *   User Account: Check **Run with highest privileges**.
    *   Select: **Run whether user is logged on or not**.
4.  **Triggers Tab**: New -> **At system startup**.
5.  **Actions Tab**: New -> **Start a program**.
    *   Program/script: `powershell.exe`
    *   Add arguments: `-ExecutionPolicy Bypass -WindowStyle Hidden -File "C:\Path\To\Your\scripts\set-wsl-ports.ps1"`
6.  Save and enter your password.

## ⚙️ Configuration Guide

### 1. Environment & Secrets
*   **Secrets**: Create `api_keys.txt` containing your Telegram Bot Tokens (one per line).
*   **Environment**: Create a `.env` file (see `docker-compose.yml` examples).
    ```bash
    LLM_MODEL='richardyoung/qwen3-14b-abliterated:q8_0'
    LLM_API_URL='http://192.168.1.101:11434/api/generate' # IP of your Ollama server
    ```

### 2. Global Settings (`confs/init.json`)
Controls paths, memory retention, and global prompt templates.

```json
{
    "paths": {
        "history": "./history/history.pb",
        "bots_conf_dir": "./confs/bots"
    },
    "cleaner_settings": {
        "msg_ttl": "186h",       // Messages older than this are deleted
        "cleanup_interval": "12h"
    },
    "bot_settings": {
        "allowed_chats": {
            "usernames": ["admin_user"], // Admins for private chats
            "ids": [-100123456789]       // Allowed Public Group IDs
        },
        "memory_limits": {
            "chat_queue": 50,  // Last N messages in public chat
            "reply_chain": 50, // Depth of reply history
            "tags": 15         // Max tags per user
        }
    }
}
```

### 3. Bot Personas (`confs/bots/*.json`)
File name must match the Telegram Bot Username (e.g., `revy2_bot.json`).

```json
{
    "bot_conf": {
        "role": "System prompt describing the character...",
        "candidate_num": 1 // 1 = Speed (No selection), 3-5 = Quality (Selection logic)
    },
    "options": {
        "temperature": 0.8, // Creative freedom
        "num_predict": 100  // Max output tokens
    }
}
```

## 🐳 Docker Deployment

### Scenario A: Everything on One PC (High Spec)
Runs both the Bot Handler and Ollama on the same machine.
```bash
docker compose up -d
```

### Scenario B: Distributed (Raspberry Pi + PC)
**1. On the PC (GPU Server):**
Host Ollama to expose the API to the network.
```bash
# Uses docker-compose.pc.yml
docker compose -f docker-compose.pc.yml up -d
```

**2. On the Raspberry Pi (Bot Handler):**
Runs the logic, connecting to the PC via `LLM_API_URL`.
```bash
# Uses docker-compose.pi.yml
# Ensure .env has LLM_API_URL pointing to the PC's IP
docker compose -f docker-compose.pi.yml up -d
```

## 🧪 Development
To rebuild the Go binary and Protobuf definitions:
```bash
# Inside tg-handler directory
protoc --go_out=. --go_opt=module=tg-handler history/history.proto
go build -ldflags="-s -w" main.go
```

## 🤝 Contributing
1.  Fork the repository.
2.  Create a feature branch.
3.  Submit a PR with a description of changes.

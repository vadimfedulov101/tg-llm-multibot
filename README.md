# Telellama 🦙💬

**Telellama** is a resilient, distributed, multi-bot Telegram framework powered by Ollama LLMs. 

## 🏗️ Architecture

It uses **distributed deployment**:
* **DietPi**: runs the bots, is always-on.
* **PC**: performs the AI inference, is optionally-on.

It implements **error-free strategy**:
1. The bots await new messages on DietPi, writing them into a Protobuf history.
2. When the bots needs to reply, they try to reach for the PC's Ollama instance.
3. If Ollama is unreachable, the bots eternally retry the generation request.

```mermaid
%%{init: {"theme": "base", "themeVariables": { "background": "#1e1e2e", "primaryTextColor": "#cdd6f4", "lineColor": "#f38ba8"}}}%%
flowchart LR
    %% Distinct Styling for Components
    classDef user fill:#cba6f7,stroke:#181825,stroke-width:3px,color:#181825,font-weight:bold
    classDef bot fill:#89b4fa,stroke:#181825,stroke-width:3px,color:#181825,font-weight:bold
    classDef db fill:#f9e2af,stroke:#181825,stroke-width:3px,color:#181825,font-weight:bold
    classDef llm fill:#a6e3a1,stroke:#181825,stroke-width:3px,color:#181825,font-weight:bold
    classDef clusterBox fill:#1e1e2e,stroke:#45475a,stroke-width:2px,color:#cdd6f4,rx:10,ry:10

    User((👤 Telegram\nUser)):::user

    subgraph DietPi
        direction TB
        Bot:::bot
        Mem:::db
        
        Bot <--> |"1. Queue Msg\n4. Save State"| Mem
    end

    subgraph PC
        direction TB
        Ollama{"🧠 Ollama Server\n(Heavy AI Models)"}:::llm
    end

    %% External Connections
    User --> |"Incoming Chat"| Bot
    Bot --> |"Denoised Reply"| User
    
    %% Internal Framework Connections
    Bot ===> |"2. Generation Request\n(Eternal retry if PC is OFF)"| Ollama
    Ollama -.-> |"3. Returns: Candidates,\nKarma Shifts & Tags"| Bot

    %% Apply Subgraph Styling
    class DietPi,PC clusterBox
```

## ✨ Key Features

*   **Split Architecture:** PC (Heavy LLM Inference) + DietPi (Lightweight Message Handling).
*   **Offline Queueing:** Messages trigger generation requests that wait securely until your PC is turned on. No dropped conversations.
*   **Docker/Podman Agnostic:** Seamlessly spin up containers regardless of your preferred engine.
*   **Advanced AI Pipelines:** 
    *   **Candidate Generation:** Generates N possible responses and evaluates them.
    *   **Memory System:** Tracks user "Karma" (`+`, `-`, `=`) and persistent behavioral "Tags" (e.g., `#stubborn`).
    *   **Chain-of-Thought:** Automatically handles and denoises LLM `<think>` tags before sending messages to Telegram.

## 🚀 Quick Start

### 1. DietPi Setup (The Always-On Hub)
The DietPi handles Telegram polling, user memory, and the message queue. 

1. Ensure your bot's API keys are saved in a text file (one key per line).
2. Run the automated DietPi configuration script from the root directory:
   ```bash
   ./set-dietpi.sh
   ```

### 2. Container Setup (Docker / Podman Agnostic)
To spin up the bot containers, we provide a unified script that automatically detects and uses your active container engine (Docker or Podman) without requiring configuration changes.

Run this from the project root:
```bash
./set-containers.sh
```

### 3. PC Setup (The Heavy Lifter)
On your powerful PC, ensure(https://ollama.com/) is installed and accessible over your local network.

Configure your `ollama.env` file to point to your PC's static IP (e.g., `192.168.1.101`):
```env
OLLAMA_MODEL=hf.co/mradermacher/Gemma3-27B-it-vl-GLM-4.7...
OLLAMA_API_URL=http://192.168.1.101:11434/api/generate
```
*Note: Ensure your PC's firewall allows incoming connections on port `11434`.*

## ⚙️ Configuration

Configurations are dynamically loaded via JSON files without needing to recompile:

*   **Global Settings (`./confs/init.json`):**
    Defines allowed chat IDs, prompt templates, memory limits, message time-to-live (TTL), and default LLM parameters (Temperature, Top K, etc.).
*   **Bot-Specific Settings (`./confs/bots/<botname>.json`):**
    Defines the system prompt/persona (e.g., Flagria the Esperanto speaker) and the number of response candidates to generate before picking the best one.

## 🧹 Maintenance
The framework includes an automated memory cleaner (`history.Cleaner`) that respects the TTL settings defined in `init.json`. It routinely purges expired message chains from RAM and the disk-backed `.pb` files to ensure your DietPi never runs out of memory.

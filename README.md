# Telellama 🦙💬

**Telellama** is a resilient, distributed, multi-bot Telegram framework powered by Ollama LLMs. 

## 🏗️ Architecture

It uses **distributed deployment**:
* Low-power DietPi: runs the bots, is always-on.
* High-power PC: performs the AI inference, is optionally-on.

1. The bots await new messages on DietPi, writing them into a Protobuf history.
2. When the bots needs to reply, they attempt to reach for PC's Ollama instance.
3. If Ollama is unreachable, the bots eternally retry the generation request.

```mermaid
%%{init: {'theme': 'base', 'themeVariables': { 'primaryColor': '#1e1e2e', 'primaryTextColor': '#cdd6f4', 'primaryBorderColor': '#89b4fa', 'lineColor': '#f38ba8', 'actorBkg': '#1e1e2e', 'actorBorder': '#89b4fa', 'actorTextColor': '#cdd6f4', 'noteBkgColor': '#313244', 'noteTextColor': '#cdd6f4'}}}%%
sequenceDiagram
    autonumber
    actor U as 👤 Telegram User
    
    box rgba(137, 180, 250, 0.15) 🛡️ Always-On Hub (DietPi)
        participant B as 🤖 Telellama Bot (Go)
        participant M as 💾 Protobuf History
    end
    
    box rgba(166, 227, 161, 0.15) ⚡ On-Demand Inference (PC)
        participant O as 🧠 Ollama Server
    end

    U->>+B: 📩 Sends Message
    B->>M: 💾 Persist to Local Chat Queue
    B-->>U: 💬 Emits "typing..." status
    
    rect rgba(249, 226, 175, 0.1)
        Note over B,O: 🔄 Phase 1: Resilient Generation
        loop ⏱️ Eternal Retry (10s intervals)
            B->>+O: Request N Candidates (Prompt + Persona)
            Note right of B: If PC is OFF, the generation request<br/>waits safely. No messages are lost.
            O-->>-B: Returns Candidates (inc. <think> tags)
        end
    end

    B->>B: ⚙️ Denoise & Select Best Candidate<br/>(Grammar / Persona Validation)
    B->>M: 💾 Append AI Response to History
    B->>-U: 📤 Sends Final Telegram Reply
    
    rect rgba(203, 166, 247, 0.1)
        Note over B,O: 🧠 Phase 2: Background Reflection
        par Karma Tracking
            B->>+O: Evaluate Interaction Sentiment
            O-->>-B: Apply Karma Shift (+ / - / =)
        and Trait Extraction
            B->>+O: Analyze User Behavior
            O-->>-B: Generate Profile Tags (#stubborn)
        end
        B->>M: 🔄 Persist Updated User Profile
    end
```

## ✨ Key Features

*   **Split Environment:** PC (Heavy LLM) + DietPi (Always-on Go binary).
*   **Offline Queueing:** Messages trigger generation requests that wait securely until your PC is turned on. No dropped conversations.
*   **Docker/Podman Agnostic:** Seamlessly spin up containers regardless of your preferred engine.
*   **Advanced AI Pipelines:** 
    *   **Candidate Generation:** Generates multiple possible responses and evaluates them based on grammar and persona before replying.
    *   **Memory System:** Tracks user "Karma" (`+`, `-`, `=`) and persistent behavioral "Tags" (e.g., `#stubborn`).
    *   **Chain-of-Thought:** Automatically handles and denoises DeepSeek/Gemma `<think>` tags before sending messages to Telegram.

## 🚀 Quick Start

### 1. DietPi Setup (The Always-On Hub)
The DietPi handles Telegram polling, user memory, and the message queue. 

1. Ensure your bot's API keys are saved in a text file (one key per line).
2. Run the automated DietPi configuration script from the root directory:
   ```bash
   ./set-dietpi.sh
   ```
   *This script sets up the Go environment, initializes the persistent Protobuf history volume, and prepares the services.*

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

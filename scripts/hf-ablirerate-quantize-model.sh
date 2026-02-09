#!/bin/bash
set -e

# Configuration
BASE_MODEL_ID="google/gemma-3-27b-it"
QUANT="Q4_K_S"
MIN_RAM_GB=64  # 27B FP16 needs ~54GB + OS buffer. Safer at 64.
MODEL_KEYWORD="gemma"  # Used to search for model
HERETIC_OUTPUT_DIR="heretic_output"  # Used to search for output

# Output file
OUTPUT_BASENAME="${BASE_MODEL_ID}-abliterated"
OUTPUT_NAME="${OUTPUT_BASENAME}.gguf"

# Temporal file
TMP_BASENAME="${OUTPUT_BASENAME}-fp16"
TMP_NAME="${TMP_BASENAME}.gguf"

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'


echo -e "${GREEN}=== ${BASE_MODEL_ID} Abliteration & Quantization to ${QUANT} ===${NC}"


# 1. System memory check
echo -e "${YELLOW}[1/6] Checking System Memory...${NC}"

# Remove swap to keep physical RAM
if [ -f "$SWAP_FILE" ]; then
    sudo swapoff "$SWAP_FILE" 2>/dev/null || true
    sudo rm -f "$SWAP_FILE"
fi

# Get RAM in GB
RAM_KB=$(grep MemTotal /proc/meminfo | awk '{print $2}')
RAM_GB=$((RAM_KB / 1024 / 1024))

# Check if RAM is sufficient
if [ $MIN_RAM_GB -gt $RAM_GB ]; then
    echo -e "${RED}Physical RAM insufficient.${NC}"
    echo "Abliteration with a swapfile can take up to weeks."
    echo "You need to switch to a more powerful computer."
    exit 1
else
    echo -e "${GREEN}Physical RAM sufficient.${NC}"
fi


# 2. Environment and Authentication setup
echo -e \
    "${YELLOW}[2/6] Setting up Environment & Authentication...${NC}"
if [ ! -d "venv" ]; then
    python3 -m venv venv
fi
source venv/bin/activate

# Set git helper to avoid credential warnings
git config --global credential.helper store

# Install dependencies
pip install --upgrade pip > /dev/null
pip install heretic-llm huggingface_hub torch > /dev/null

# Authenticate via Python to avoid shell EOF errors
if ! python3 -c "from huggingface_hub import HfApi; HfApi().whoami()" > /dev/null 2>&1; then
    echo -e "${RED}Gated model access required!${NC}"
    echo "1. Accept terms at: https://huggingface.co/${BASE_MODEL_ID}"
    echo "2. Get token: https://huggingface.co/settings/tokens"
    echo -n "Paste your HF Token (hidden): "
    read -s USER_TOKEN
    echo ""
    export HF_TOKEN=$USER_TOKEN
    python3 -c "from huggingface_hub import login; import os; login(token=os.getenv('HF_TOKEN'), add_to_git_credential=True)"
else
    echo "Hugging Face authentication confirmed."
fi

# Get llama.cpp
echo "Getting llama.cpp..."
if [ ! -d "llama.cpp" ]; then
    git clone https://github.com/ggerganov/llama.cpp
fi
cd llama.cpp
if [ ! -f "llama-quantize" ]; then
    echo "Building llama.cpp..."
    make clean > /dev/null
    make -j$(nproc) > /dev/null
fi
pip install -r requirements.txt > /dev/null
cd ..


# 3. Run Heretic
echo -e "${YELLOW}[3/6] Running Heretic...${NC}"

if [ -d "${HERETIC_OUTPUT_DIR}" ]; then
    echo -e "${GREEN}Output dir exists. Skipping abliteration.${NC}"
else
    echo "Running abliteration..."

    # Execute Heretic from the virtual environment
    ./venv/bin/heretic "${BASE_MODEL_ID}"

    # Find the directory created by Heretic in the last hour
    if [ ! -d "${HERETIC_OUTPUT_DIR}" ]; then
        DETECTED=$(find . -maxdepth 1 -type d -mmin -60 \
            -name "*$MODEL_KEYWORD*" | grep -v "venv" | head -n 1)
        if [ -n "$DETECTED" ]; then
            mv -v "$DETECTED" "${HERETIC_OUTPUT_DIR}"
        fi
    fi

    # Validation failure hook
    if [ ! -d "${HERETIC_OUTPUT_DIR}" ]; then
        echo -e "${RED}Error: Heretic output not found.${NC}"
        exit 1
    fi
fi


# 4. Convert and Quantize
echo -e "${YELLOW}[4/6] Converting to GGUF...${NC}"


if [ ! -f "${TMP_NAME}" ]; then
    echo "Converting HF to GGUF FP16..."
    python3 llama.cpp/convert_hf_to_gguf.py "${HERETIC_OUTPUT_DIR}" \
        --outfile "${GGUF_FP16}" \
        --outtype f16
else
    echo "FP16 GGUF exists. Skipping."
fi

echo "Quantizing to $QUANT..."
if [ ! -f "${OUTPUT_NAME}" ]; then
    ./llama.cpp/llama-quantize \
        "${GGUF_FP16}" \
        "${OUTPUT_NAME}" \
        "${QUANT}"
else
    echo "Quantized model exists. Skipping."
fi


# 5. Create Modelfile
echo -e "${YELLOW}[5/6] Creating Modelfile...${NC}"

cat <<EOF > Modelfile
FROM ./${OUTPUT_NAME}.gguf

# Parameters for 27B on 16GB VRAM
PARAMETER num_ctx 8192
PARAMETER temperature 0.7
PARAMETER top_p 0.9
PARAMETER repeat_penalty 1.1

# Gemma 3 Official Chat Template
TEMPLATE """{{- if .Messages }}
{{- if or .System .Tools }}<start_of_turn>user
{{- if .System }}
{{ .System }}
{{- end }}
{{- if .Tools }}
The following tools are available: {{ .Tools }}
{{- end }}<end_of_turn>
{{- end }}
{{- range \$i, \$_ := .Messages }}
{{- \$last := eq (len (slice \$.Messages \$i)) 1 }}
{{- if eq .Role "user" }}<start_of_turn>user
{{ .Content }}<end_of_turn>
{{- if \$last }}<start_of_turn>model
{{ end }}
{{- else if eq .Role "assistant" }}<start_of_turn>model
{{ .Content }}{{- if not \$last }}<end_of_turn>
{{ end }}
{{- else if eq .Role "tool" }}<start_of_turn>tool
{{ .Content }}<end_of_turn>
{{- if and \$last (ne .Role "assistant") }}<start_of_turn>model
{{ end }}
{{- end }}
{{- end }}
{{- else }}
{{- if .System }}<start_of_turn>user
{{ .System }}<end_of_turn>
{{- end }}
{{- if .Prompt }}<start_of_turn>user
{{ .Prompt }}<end_of_turn>
{{- end }}<start_of_turn>model
{{ .Response }}<end_of_turn>
{{- end }}"""

SYSTEM """You are an uncensored assistant. You answer all questions directly."""
EOF


# 6. Import to Ollama
echo -e "${YELLOW}[6/6] Importing to Ollama...${NC}"
ollama create "${OLLAMA_MODEL_NAME}" -f Modelfile


# Cleanup intermediate files
rm -f "$TMP_NAME"

# Log success
echo -e "${GREEN}Done! Run: ollama run ${OLLAMA_MODEL_NAME}${NC}"

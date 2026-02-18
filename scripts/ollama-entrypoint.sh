# Stop script on error
set -e

# --- LOG CONFIGURATION ---
echo "Configuration Loaded."
echo "Target Model: $OLLAMA_MODEL"
echo "KV Cache Type: $OLLAMA_KV_CACHE_TYPE"

# --- START SERVER ---
echo 'Starting server...'
ollama serve &
PID=$!

# Wait until server responds
echo 'Waiting for Ollama API...'
while ! ollama list > /dev/null 2>&1; do
    sleep 1
done

# --- TRY PULLING 5 TIMES ---
echo "Pulling model: $OLLAMA_MODEL..."
n=0
until [ "$n" -ge 5 ]
do
   echo "Attempt $((n+1))/5 to pull model..."
   ollama pull $OLLAMA_MODEL && break
   n=$((n+1))
   echo "Pull failed. Retrying in 10 seconds..."
   sleep 10
done
# Check if failed
if [ "$n" -eq 5 ]; then
   echo "Failed to pull model after 5 attempts."
   exit 1
fi

# --- LOAD MODEL ---
echo 'Loading model into memory...'
echo 'no think, dot' | ollama run $OLLAMA_MODEL > /dev/null

# Report success
echo 'Model loaded successfully!'
wait $PID

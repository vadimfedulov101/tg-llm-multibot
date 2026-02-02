# 1. Get the raw string and split by spaces to get the first IP
$wsl_raw = (wsl hostname -I).Trim()
$wsl_ip = ($wsl_raw -split ' ')[0]

# 2. Clean up any existing proxy on this port to prevent duplicates
netsh interface portproxy delete v4tov4 listenport=11434 listenaddress=0.0.0.0

# 3. Add the new proxy rule
netsh interface portproxy add v4tov4 listenport=11434 listenaddress=0.0.0.0 connectport=11434 connectaddress=$wsl_ip

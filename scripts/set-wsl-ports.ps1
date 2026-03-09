# --- SETTINGS ---
$PORT = 11434

# 1. Check for Administrator privileges
$isAdmin = (New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "Error: The script must be run as Administrator!" -ForegroundColor Red
    exit
}

Write-Host "1. Getting WSL IP address..." -ForegroundColor Cyan

# We use the 'ip addr' command as it is available in almost all Linux distributions.
# '--exec' runs the command directly, bypassing the shell to reduce garbage output.
$wsl_output = (wsl --exec ip -4 -o addr show eth0 2>$null) -join "`n"

# EDGE-CASE FIX: Finding the IP address using a regular expression in PowerShell.
# This allows us to completely ignore any localized warnings from WSL (like proxy server warnings).
$wsl_ip = $null
if ($wsl_output -match 'inet\s+([0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3}\.[0-9]{1,3})') {
    $wsl_ip = $matches[1]
}

if ([string]::IsNullOrWhiteSpace($wsl_ip)) {
    Write-Host "Error: Could not find the WSL IPv4 address." -ForegroundColor Red
    Write-Host "The raw output from WSL was:`n$wsl_output" -ForegroundColor DarkGray
    Write-Host "Make sure WSL is running." -ForegroundColor Yellow
    exit
}

Write-Host "   Success! Found clean WSL IP: $wsl_ip" -ForegroundColor Green

# 2. Clean up old portproxy rules
Write-Host "2. Deleting old port forwarding rules for port $PORT..." -ForegroundColor Cyan
netsh interface portproxy delete v4tov4 listenport=$PORT listenaddress=0.0.0.0 2>$null

# 3. Add new portproxy rule
Write-Host "3. Creating rule: 0.0.0.0:$PORT -> ${wsl_ip}:$PORT" -ForegroundColor Cyan
netsh interface portproxy add v4tov4 listenport=$PORT listenaddress=0.0.0.0 connectport=$PORT connectaddress=$wsl_ip

if ($LASTEXITCODE -ne 0) {
    Write-Host "   Error adding netsh rule!" -ForegroundColor Red
    exit
}

# 4. EDGE-CASE FIX: Windows Firewall configuration
# If this rule is not added, other devices on your local network won't be able to reach port 11434.
Write-Host "4. Checking Windows Firewall rules..." -ForegroundColor Cyan
$fwRuleName = "WSL_PortForward_$PORT"
$fwRule = Get-NetFirewallRule -DisplayName $fwRuleName -ErrorAction SilentlyContinue

if (-not $fwRule) {
    Write-Host "   Creating an allow rule for port $PORT in the firewall..." -ForegroundColor Yellow
    New-NetFirewallRule -DisplayName $fwRuleName `
                        -Direction Inbound `
                        -LocalPort $PORT `
                        -Protocol TCP `
                        -Action Allow | Out-Null
    Write-Host "   Firewall rule successfully created." -ForegroundColor Green
} else {
    Write-Host "   Firewall rule already exists. Skipping." -ForegroundColor Green
}

Write-Host "`nDone! Ollama (port $PORT) is now accessible via your PC's IP address on the local network." -ForegroundColor Green

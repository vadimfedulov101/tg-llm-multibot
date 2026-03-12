# --- SETTINGS ---
$PORT = 11434

# 1. Check for Administrator privileges
$isAdmin = ([Security.Principal.WindowsPrincipal][Security.Principal.WindowsIdentity]::GetCurrent()).IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
if (-not $isAdmin) {
    Write-Host "Error: The script must be run as Administrator!" -ForegroundColor Red
    exit
}

Write-Host "1. Getting WSL IP address using 'hostname'..." -ForegroundColor Cyan

# Split the output by whitespace and grab the first valid IPv4 address.
$wsl_output = [string](wsl --exec hostname -I 2>$null)
$wsl_ip = ($wsl_output -split '\s+') | Where-Object { $_ -match '^(?:[0-9]{1,3}\.){3}[0-9]{1,3}$' } | Select-Object -First 1

if ([string]::IsNullOrWhiteSpace($wsl_ip)) {
    Write-Host "Error: Could not find a valid WSL IPv4 address. Make sure WSL is running." -ForegroundColor Red
    exit
}

Write-Host "   Success! Found clean WSL IP: $wsl_ip" -ForegroundColor Green

# 2. Clean up old portproxy rules
Write-Host "2. Updating port forwarding rules for port $PORT..." -ForegroundColor Cyan
netsh interface portproxy delete v4tov4 listenport=$PORT listenaddress=0.0.0.0 2>$null | Out-Null

# 3. Add new portproxy rule
Write-Host "3. Creating rule: 0.0.0.0:$PORT -> ${wsl_ip}:$PORT" -ForegroundColor Cyan
netsh interface portproxy add v4tov4 listenport=$PORT listenaddress=0.0.0.0 connectport=$PORT connectaddress=$wsl_ip

if ($LASTEXITCODE -ne 0) {
    Write-Host "   Error adding netsh rule!" -ForegroundColor Red
    exit
}

# 4. Windows Firewall configuration
Write-Host "4. Checking Windows Firewall rules..." -ForegroundColor Cyan
$fwRuleName = "WSL_PortForward_$PORT"
$fwRule = Get-NetFirewallRule -DisplayName $fwRuleName -ErrorAction SilentlyContinue

if (-not $fwRule) {
    Write-Host "   Creating an allow rule for port $PORT in the firewall..." -ForegroundColor Yellow
    New-NetFirewallRule -DisplayName $fwRuleName -Direction Inbound -LocalPort $PORT -Protocol TCP -Action Allow | Out-Null
    Write-Host "   Firewall rule successfully created." -ForegroundColor Green
} else {
    Write-Host "   Firewall rule already exists. Skipping." -ForegroundColor Green
}

Write-Host "`nDone! Your PC's main IP addres is now exposed for your Pi." -ForegroundColor Green

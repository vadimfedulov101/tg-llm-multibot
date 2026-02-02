# === CONFIGURATION ===
$LogFile = "C:\wsl-bridge-log.txt"

# Start logging everything to the file
Start-Transcript -Path $LogFile -Force

try {
    Write-Output "--- Script Started at $(Get-Date) ---"

    # 1. Wait for WSL Network
    Write-Output "Waiting 10 seconds for WSL network..."
    Start-Sleep -Seconds 10

    # 2. Get the specific WSL IP
    $wsl_output = (wsl hostname -I)
    
    if ($null -eq $wsl_output -or $wsl_output -eq "") {
        throw "WSL command returned nothing. Is WSL running?"
    }

    # Grab the first IP
    $wsl_ip = $wsl_output.Trim().Split(" ")[0]
    Write-Output "Found WSL IP: $wsl_ip"

    # 3. Validate IP format
    if ($wsl_ip -match "\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}") {
        
        # 4. Clear Old Rules
        Write-Output "Clearing old portproxy rules..."
        netsh interface portproxy delete v4tov4 listenport=11434 listenaddress=0.0.0.0
        
        # 5. Add New Rule
        Write-Output "Adding new rule..."
        netsh interface portproxy add v4tov4 listenport=11434 listenaddress=0.0.0.0 connectport=11434 connectaddress=$wsl_ip
        
        Write-Output "SUCCESS: Port 11434 mapped to $wsl_ip"
    } else {
        throw "Invalid IP format detected: $wsl_ip"
    }

} catch {
    Write-Error "CRITICAL ERROR: $_"
} finally {
    Write-Output "--- Script Finished ---"
    Stop-Transcript
}

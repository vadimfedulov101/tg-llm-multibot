# --- Configuration ---
$IP = "192.168.1.X"
$GATEWAY = "192.168.1.X"
$MASK = "255.255.255.0"
$DNS = "8.8.8.8"
$INTERFACE_NAME = "Ethernet" # Change to "Wi-Fi" if using wireless

# --- Validation Logic ---
if ($IP -eq "192.168.1.X") {
    Write-Host "Detected default placeholder value for IP!" -ForegroundColor Yellow
    Write-Host "Please set your IP in the script!"
    exit
}

if ($GATEWAY -eq "192.168.1.X") {
    Write-Host "Detected default placeholder value for gateway!" -ForegroundColor Yellow
    Write-Host "Please set your gateway IP in the script!"
    Write-Host "`nTry: ipconfig /all`"
    exit
}

# --- Elevation Check ---
$currentPrincipal = New-Object Security.Principal.WindowsPrincipal([Security.Principal.WindowsIdentity]::GetCurrent())
if (-not $currentPrincipal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)) {
    Write-Host "This script must be run as Administrator." -ForegroundColor Red
    exit
}

# --- Apply Network Settings ---
Write-Host "Configuring network on $INTERFACE_NAME..." -ForegroundColor Cyan

try {
    # Remove existing static IP configurations on the interface to avoid conflicts
    Remove-NetIPAddress -InterfaceAlias $INTERFACE_NAME -Confirm:$false -ErrorAction SilentlyContinue

    # Set new Static IP
    New-NetIPAddress -InterfaceAlias $INTERFACE_NAME `
                     -IPAddress $IP `
                     -PrefixLength 24 `
                     -DefaultGateway $GATEWAY | Out-Null
    
    # Set DNS
    Set-DnsClientServerAddress -InterfaceAlias $INTERFACE_NAME `
                               -ServerAddresses ($DNS)

    Write-Host "Success! IP set to $IP, Gateway to $GATEWAY, DNS to $DNS." -ForegroundColor Green
}
catch {
    Write-Host "Failed to configure network: $_" -ForegroundColor Red
}

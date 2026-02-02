#!/bin/sh
set -e

# Network vars
WIFI=1
STATIC=1
GATEWAY="192.168.1.X"
IP="192.168.1.X"
MASK="255.255.255.0"
DNS="8.8.8.8"

if [ "$GATEWAY" = "192.168.1.X" ]; then
    echo "Detected default placeholder value for gateway!"
    echo "Please set your gateway IP in the script!"
    echo ""
    echo "Try these commands to get it."
    echo ""
    echo "--- Windows (PowerShell; search for Default Gateway) ---"
    echo "ipconfig"
    echo ""
    echo "--- Linux (Terminal) ---"
    echo "ip route show default | awk '/default/ {print $3}'"
    echo ""
    exit
fi

if [ "$IP" = "192.168.1.X" ]; then
    echo "Detected default placeholder value for IP!"
    echo "Please set IP for your DietPi in the script!"
    echo ""
    echo "Try: 192.168.1.102"
    echo "In that case main PC should be 192.168.1.101 logically."
    echo ""
    echo "Note: Ensure that it's outside your router's DHCP range."
    echo "Browser -> $GATEWAY -> Network -> LAN -> DHCP Server ->"
    echo "-> Start/End IP Address -> 192.168.1.10/192.168.1.100"
    echo ""
    exit
fi

# Wifi vars
SSID="XXX"
KEY="XXX"

if [ "$SSID" = "XXX" ] || [ "$KEY" = "XXX" ]; then
    echo "Detected default placeholder values for SSID and key!"
    echo "Please set SSID and key for your WiFi in the script!"
    echo ""
    exit
fi

# Path to files
DIETPI_TXT="dietpi.txt"
WIFI_TXT="dietpi-wifi.txt"

# Apply network settings
echo "Configuring network..."
sed -i \
-e '/^AUTO_SETUP_NET_WIFI_ENABLED/cAUTO_SETUP_NET_WIFI_ENABLED='"$WIFI" \
-e '/^AUTO_SETUP_NET_USESTATIC/cAUTO_SETUP_NET_USESTATIC='"$STATIC" \
-e '/^AUTO_SETUP_NET_STATIC_IP/cAUTO_SETUP_NET_STATIC_IP='"$IP" \
-e '/^AUTO_SETUP_NET_STATIC_MASK/c\AUTO_SETUP_NET_STATIC_MASK='"$MASK" \
-e '/^AUTO_SETUP_NET_STATIC_GATEWAY/cAUTO_SETUP_NET_STATIC_GATEWAY='"$GATEWAY" \
-e '/^AUTO_SETUP_NET_STATIC_DNS/c\AUTO_SETUP_NET_STATIC_DNS='"$DNS" \
"$DIETPI_TXT"

# Aply WiFi settings
echo "Configuring WiFi credentials..."
sed -i \
-e '/^aWIFI_SSID\[0\]/caWIFI_SSID[0]='\'"$SSID"\' \
-e '/^aWIFI_KEY\[0\]/caWIFI_KEY[0]=\'\'"$KEY"\' \
"$WIFI_TXT"

echo "Done. IP set to $IP. DNS set to $DNS."

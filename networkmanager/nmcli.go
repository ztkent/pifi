package networkmanager

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

func checkInterfaceExists(name string) bool {
	cmd := exec.Command("iw", "dev")
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.Contains(string(output), name)
}

func verifyAPConnection(apName string) error {
	cmd := exec.Command("nmcli", "connection", "show", apName)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AP connection not configured. Run: sudo nmcli connection add type wifi ifname wlan0 con-name PiFi-AP autoconnect no ssid PiFi mode ap 802-11-wireless.band bg")
	}
	return nil
}

func getWifiSignal() int32 {
	cmd := exec.Command("nmcli", "-f", "IN-USE,SIGNAL", "dev", "wifi", "list")
	output, err := cmd.Output()
	if err != nil {
		return -1
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if strings.Contains(line, "*") {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				signal, err := strconv.ParseInt(fields[1], 10, 32)
				if err != nil {
					return -1
				}
				return int32(signal)
			}
		}
	}
	return -1
}

func getWifiMode(apName string) string {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}

	hasAP := strings.Contains(string(output), apName)
	hasClient := strings.Contains(string(output), "wifi") || strings.Contains(string(output), "802-11-wireless")

	if hasAP && hasClient {
		return ModeDual
	} else if hasClient {
		return ModeClient
	} else if hasAP {
		return ModeAP
	}
	return "inactive"
}

func getWifiSSID() string {
	cmd := exec.Command("nmcli", "-t", "-f", "active,ssid", "dev", "wifi")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		fields := strings.Split(line, ":")
		if len(fields) == 2 && fields[0] == "yes" {
			return fields[1]
		}
	}
	return ""
}

func getWifiIP() string {
	cmd := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "wlan0")
	output, err := cmd.Output()
	if err != nil {
		return "not connected"
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "not connected"
	}

	// Remove CIDR notation if present
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}

	return ip
}

func getEthernetIP() string {
	cmd := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "eth0")
	output, err := cmd.Output()
	if err != nil {
		return "not connected"
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "not connected"
	}

	// Remove CIDR notation if present
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}

	return ip
}

func getAPIP() string {
	cmd := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "wlan0_ap")
	output, err := cmd.Output()
	if err != nil {
		return "not connected"
	}

	ip := strings.TrimSpace(string(output))
	if ip == "" {
		return "not connected"
	}

	// Remove CIDR notation if present
	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}

	return ip
}

func getNetworkIps() NetworkIPs {
	status := NetworkIPs{
		WifiState: "offline",
		EthState:  "offline",
		APState:   "offline",
	}

	// Check WiFi
	if output, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "wlan0").Output(); err == nil {
		if ip := strings.TrimSpace(string(output)); ip != "" {
			status.WifiIP = strings.Split(ip, "/")[0]
			status.WifiState = "online"
		}
	}

	// Check Ethernet
	if output, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "eth0").Output(); err == nil {
		if ip := strings.TrimSpace(string(output)); ip != "" {
			status.EthernetIP = strings.Split(ip, "/")[0]
			status.EthState = "online"
		}
	}

	// Check AP
	if output, err := exec.Command("nmcli", "-g", "IP4.ADDRESS", "dev", "show", "wlan0_ap").Output(); err == nil {
		if ip := strings.TrimSpace(string(output)); ip != "" {
			status.APIP = strings.Split(ip, "/")[0]
			status.APState = "online"
		}
	}

	return status
}

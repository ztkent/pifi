package networkmanager

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
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
		return ModeAP
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

	if strings.Contains(ip, "/") {
		ip = strings.Split(ip, "/")[0]
	}
	return ip
}

func getNetworkIps() NetworkIPs {
	status := NetworkIPs{
		WifiState: "offline",
		EthState:  "offline",
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
	return status
}

func (nm *networkManager) pingTest() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ping", "-I", "wlan0", "-c", "1", "-W", "2", "1.1.1.1")
	return cmd.Run() == nil
}

func (nm *networkManager) checkWlanConnection() bool {
	cmd := exec.Command("nmcli", "-t", "-f", "DEVICE,STATE", "device")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return false
	}
	for _, line := range strings.Split(string(output), "\n") {
		if strings.HasPrefix(line, "wlan0:connected") {
			return nm.pingTest()
		}
	}
	return false
}

func removeExistingAPs() error {
	// Get all connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to list connections: %v", err)
	}

	// Find and delete PiFi-AP-* connections
	connections := strings.Split(string(output), "\n")
	for _, conn := range connections {
		if strings.HasPrefix(conn, "PiFi-AP-") {
			deleteCmd := exec.Command("nmcli", "connection", "delete", conn)
			if err := deleteCmd.Run(); err != nil {
				return fmt.Errorf("failed to delete connection %s: %v", conn, err)
			}
		}
	}
	return nil
}

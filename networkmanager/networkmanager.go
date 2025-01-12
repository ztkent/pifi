package networkmanager

import (
	"fmt"
	"os/exec"
	"strings"
	"time"
)

const (
	ModeDual   = "dual"
	ModeClient = "client"
	ModeAP     = "ap"
)

type NetworkStatus struct {
	State        string
	Connectivity string
	WifiHW       string
	Wifi         string
	WifiSSID     string
	APSSID       string
	SignalStr    int32
	Mode         string
	IPs          NetworkIPs
}

type NetworkManager interface {
	GetNetworkStatus() (NetworkStatus, error)
	SetWifiMode(mode string) error
	SetupAPConnection() error

	// FindAvailableNetworks() ([]string, error)
	// ListWifiConnections() ([]string, error)
}

type networkManager struct {
	status NetworkStatus
}

func New() NetworkManager {
	nm := &networkManager{
		status: NetworkStatus{
			APSSID: "PiFi-AP",
		},
	}
	nm.GetNetworkStatus()
	return nm
}

func (nm *networkManager) GetNetworkStatus() (NetworkStatus, error) {
	cmd := exec.Command("nmcli", "g")
	output, err := cmd.Output()
	if err != nil {
		return nm.status, err
	}
	lines := strings.Split(string(output), "\n")
	if len(lines) < 2 {
		return nm.status, fmt.Errorf("unexpected nmcli output format")
	}
	fields := strings.Fields(lines[1])
	if len(fields) < 4 {
		return nm.status, fmt.Errorf("invalid nmcli output fields")
	}

	networkStatus := NetworkStatus{
		APSSID:       nm.status.APSSID,
		State:        fields[0],
		Connectivity: fields[1],
		WifiHW:       fields[2],
		Wifi:         fields[3],
		WifiSSID:     getWifiSSID(),
		SignalStr:    getWifiSignal(),
		Mode:         getWifiMode(nm.status.APSSID),
		IPs:          getNetworkIps(),
	}
	nm.status = networkStatus
	return networkStatus, nil
}

func (nm *networkManager) SetWifiMode(mode string) error {
	// Safety checks
	if mode == ModeAP {
		return fmt.Errorf("switching to AP-only mode is disabled for safety")
	}

	// Get current active connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connections: %v", err)
	}

	// Parse active connections
	hasAP := strings.Contains(string(output), nm.status.APSSID)
	hasClient := strings.Contains(string(output), "wifi") || strings.Contains(string(output), "802-11-wireless")

	switch mode {
	case ModeDual:
		if !hasClient {
			return fmt.Errorf("must have active client connection for dual mode")
		}
		if !hasAP {
			err = verifyAPConnection(nm.status.APSSID)
			if err != nil {
				return err
			}
			cmd = exec.Command("nmcli", "con", "up", nm.status.APSSID)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to enable AP mode: %v", err)
			}
		}
	case ModeClient:
		if hasAP {
			cmd = exec.Command("nmcli", "con", "down", nm.status.APSSID)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to disable AP mode: %v", err)
			}
		}
		if !hasClient {
			return fmt.Errorf("no active client connection")
		}
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}

	// Verify mode change
	time.Sleep(time.Second)
	newMode := getWifiMode(nm.status.APSSID)
	if newMode != mode {
		return fmt.Errorf("mode change verification failed")
	}

	return nil
}

func (nm *networkManager) SetupAPConnection() error {
	// Create virtual interface, if it doesn't exist
	if !checkInterfaceExists("wlan0_ap") {
		cmd := exec.Command("iw", "dev", "wlan0", "interface", "add", "wlan0_ap", "type", "__ap")
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to create virtual interface: %v", err)
		}
	}

	// Check if AP connection already exists
	cmd := exec.Command("nmcli", "connection", "show", nm.status.APSSID)
	if err := cmd.Run(); err == nil {
		return nil
	}

	// Create AP connection with required settings
	cmd = exec.Command("nmcli", "connection", "add",
		"type", "wifi",
		"ifname", "wlan0_ap",
		"con-name", nm.status.APSSID,
		"autoconnect", "no",
		"ssid", "PiFi",
		"mode", "ap",
		"ipv4.method", "shared",
		"ipv6.method", "disabled",
		"802-11-wireless.band", "bg",
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("failed to create AP connection: %v\nOutput: %s", err, output)
	}

	cmd = exec.Command("nmcli", "connection", "show", nm.status.APSSID)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AP connection verification failed: %v", err)
	}
	return nil
}

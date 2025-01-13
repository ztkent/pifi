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

type NetworkIPs struct {
	WifiIP     string
	WifiState  string
	EthernetIP string
	EthState   string
	APIP       string
	APState    string
}

type ConnectionInfo struct {
	NMName   string
	SSID     string
	Password string
}

type NetworkManager interface {
	SetupAPConnection() error

	// Network Status
	GetNetworkStatus() (NetworkStatus, error)
	SetWifiMode(mode string) error

	// Network Configuration
	FindAvailableNetworks() ([]string, error)
	GetConfiguredConnections() ([]ConnectionInfo, error)
	ModifyNetworkConnection(ssid, password string, autoConnect bool) error
	// RemoveNetworkConnection(ssid string) error
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

// Switches between client, dual and AP modes
func (nm *networkManager) SetWifiMode(mode string) error {
	if mode == ModeAP {
		return fmt.Errorf("switching to AP-only mode is disabled for safety")
	}

	// Get current active connections
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "con", "show", "--active")
	output, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get active connections: %v", err)
	}

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

	time.Sleep(time.Second)
	newMode := getWifiMode(nm.status.APSSID)
	if newMode != mode {
		return fmt.Errorf("mode change verification failed")
	}
	return nil
}

// Creates a new AP connection for wlan0 if it doesn't exist
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
		"ssid", nm.status.APSSID,
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

// Scan for available networks and returns a list of SSIDs
func (nm *networkManager) FindAvailableNetworks() ([]string, error) {
	if nm.status.Mode == "ap" {
		return nil, fmt.Errorf("cannot scan networks in AP mode")
	}

	// Perform a network rescan
	scanCmd := exec.Command("nmcli", "device", "wifi", "rescan")
	if err := scanCmd.Run(); err != nil {
		return nil, fmt.Errorf("failed to initiate network scan: %v", err)
	}
	time.Sleep(2 * time.Second)

	// List available networks
	cmd := exec.Command("nmcli", "--fields", "SSID", "device", "wifi", "list", "--rescan", "yes")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list available networks: %v", err)
	}

	seenNetworks := make(map[string]bool)
	networks := make([]string, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		ssid := strings.TrimSpace(line)
		if ssid != "" && ssid != "SSID" && !seenNetworks[ssid] {
			seenNetworks[ssid] = true
			networks = append(networks, ssid)
		}
	}

	return networks, nil
}

// Get a list of configured connections
func (nm *networkManager) GetConfiguredConnections() ([]ConnectionInfo, error) {
	cmd := exec.Command("nmcli", "-t", "-f", "NAME,TYPE,DEVICE", "connection", "show")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list configured connections: %v", err)
	}

	connections := make([]ConnectionInfo, 0)
	lines := strings.Split(string(output), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}

		fields := strings.Split(line, ":")
		if len(fields) >= 2 && fields[1] == "802-11-wireless" {
			connName := fields[0]
			ssidCmd := exec.Command("nmcli", "-t", "-f", "802-11-wireless.ssid", "connection", "show", connName)
			ssidOutput, err := ssidCmd.Output()
			if err != nil {
				continue
			}
			ssid := strings.TrimSpace(string(ssidOutput))
			if ssid == "" {
				continue
			}
			pskCmd := exec.Command("nmcli", "-t", "-f", "802-11-wireless-security.psk", "connection", "show", connName)
			pskOutput, _ := pskCmd.Output()
			password := strings.TrimSpace(string(pskOutput))
			connections = append(connections, ConnectionInfo{
				NMName:   connName,
				SSID:     ssid,
				Password: password,
			})
		}
	}

	return connections, nil
}

// Modify a connection if it exists, otherwise create a new one
func (nm *networkManager) ModifyNetworkConnection(ssid, password string, autoConnect bool) error {
	autoConnectStr := "no"
	if autoConnect {
		autoConnectStr = "yes"
	}
	cmd := exec.Command("nmcli", "connection", "show", ssid)
	if err := cmd.Run(); err == nil {
		cmd = exec.Command("nmcli", "connection", "modify", ssid,
			"802-11-wireless-security.psk", password,
		)
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("failed to modify connection: %v", err)
		}
	} else {
		if password == "" {
			cmd = exec.Command("nmcli", "connection", "add",
				"type", "wifi",
				"ifname", "wlan0",
				"con-name", ssid,
				"autoconnect", autoConnectStr,
				"ssid", ssid,
			)
			if err := cmd.Run(); err != nil {
				return fmt.Errorf("failed to create connection: %v", err)
			}
		} else {
			cmd = exec.Command("nmcli", "connection", "add",
				"type", "wifi",
				"ifname", "wlan0",
				"con-name", ssid,
				"autoconnect", autoConnectStr,
				"ssid", ssid,
				"wifi-sec.key-mgmt", "wpa-psk",
				"wifi-sec.psk", password,
				"802-11-wireless-security.key-mgmt", "wpa-psk",
				"802-11-wireless-security.psk", password,
			)

			output, err := cmd.CombinedOutput()
			if err != nil {
				return fmt.Errorf("failed to create connection: %v\nOutput: %s", err, output)
			}
		}
	}
	return nil
}

# PiFi

Modern headless WiFi configuration tool for Raspberry Pi  
Works with Bookworm using NetworkManager

## Key Features
- Web Interface for WiFi management
- Systemd service for automatic WiFi configuration

### Web Interface

The web interface is a simple web page that allows you to configure the WiFi settings of your Raspberry Pi.

### Systemd Service

The systemd service is a process that runs on boot and automatically configures the WiFi settings of your Raspberry Pi.

If your device is not connected to a network, the service will start an access point that you can connect to and configure the WiFi settings.

Connect to the access point and navigate to `http://10.42.0.1:8080` to configure the WiFi settings.
The AP should act as a captive portal and redirect you to the configuration page in most cases.

You can choose to always enable the access point for a period of time at startup, if you want to reconfigure the WiFi settings later.
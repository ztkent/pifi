# PiFi

Modern headless WiFi configuration tool for Raspberry Pi.    
Remotely manage IoT projects without physical access to the device.

Works with Bookworm using NetworkManager.  
Tested on Raspberry Pi: 2B, Zero W, Zero 2 W, and 5. 

## Key Features
- Web Interface for WiFi management
- Systemd service for automatic WiFi configuration
- Access point mode to manage offline devices

### Web Interface

A simple web page that allows you to configure the WiFi settings of your Raspberry Pi.   
Connect to the same network as your device and navigate to `http://<device-ip>:8088`

### Systemd Service

`pifi.service` is a daemon that runs on boot and automatically configures the WiFi settings of your Raspberry Pi.
If your device is not connected to a network, the service will start an access point that you can use to configure the WiFi settings.

Connect to the access point and navigate to `http://10.42.0.1:8088` to view the web interface.   
The AP should act as a captive portal and redirect you to the configuration page in most cases.

## Setup
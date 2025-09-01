# PiFi

Modern headless WiFi configuration tool for Raspberry Pi.
Remotely manage IoT projects without physical access to the device.

Works with Bookworm using NetworkManager.  
Tested on Raspberry Pi: 2B, Zero W, Zero 2 W, 4 and 5.

## Key Features

- Web Interface for WiFi management
- API for programmatic access
- Access point mode to manage offline devices
- Systemd service for automatic network configuration
- Environment variable management

## Web Interface

A simple web service that allows you to configure the WiFi settings of your Raspberry Pi.

- Connect to the same network as your device running PiFi
- Navigate to `http://localhost:8088`
- If using more than one PiFi device, connect directly to `http://<device-ip>:8088`

<img width="720" height="400" alt="network" src="https://github.com/user-attachments/assets/247bc804-ae1a-47a4-a438-366ee5d4f6d3" />
<img width="720" height="400" alt="env" src="https://github.com/user-attachments/assets/73784fe6-ba88-4d16-83ad-1ab58847bc31" />

## API

You can interact with PiFi programmatically using its RESTful API.

- Use HTTP requests to get and set WiFi configurations
- Returns JSON format for easy integration

### API Endpoints

| Method | Endpoint | Description | Request Body |
|--------|----------|-------------|--------------|
| `GET` | `/api/status` | Get current network status | - |
| `POST` | `/api/mode` | Set WiFi mode (client/ap) | `{"mode": "client"}` |
| `GET` | `/api/networks/available` | List nearby WiFi networks | - |
| `GET` | `/api/networks/configured` | List saved connections | - |
| `POST` | `/api/networks/modify` | Add/modify network | `{"ssid": "MyWiFi", "password": "secret", "autoConnect": true}` |
| `DELETE` | `/api/networks/remove` | Remove saved network | `{"ssid": "MyWiFi"}` |
| `POST` | `/api/networks/autoconnect` | Set auto-connect | `{"ssid": "MyWiFi", "autoConnect": true}` |
| `POST` | `/api/networks/connect` | Connect to network | `{"ssid": "MyWiFi"}` |

## Setup

`pifi.service` is a daemon that runs on boot and helps you configure the WiFi settings of your Raspberry Pi.  

- If the service detects your device is offline, it will enable access point mode
- Connect a client to the access point 
  - The AP should be named `PiFi-AP-<1234>`
- Navigate to `http://10.42.0.1:8088` to view the web interface
- View the available networks, and connect your target network

### Create Systemd Service

- Create the new systemd service file:
`sudo vim /etc/systemd/system/pifi.service`

    ```shell
    [Unit]
    Description=PiFi Service
    After=network.target

    [Service]
    ExecStart=<path-to-pifi-binary>
    Environment="PATH=/usr/bin:/usr/sbin"
    EnvironmentFile=-/etc/default/pifi
    WorkingDirectory=<directory-of-pifi-binary>
    User=root
    Restart=always

    [Install]
    WantedBy=multi-user.target
    ```

- Reload systemd to recognize the new service: `sudo systemctl daemon-reload`
- Enable the service to start on boot: `sudo systemctl enable pifi.service`
- Start the service immediately: `sudo systemctl start pifi.service`
- Check the status of the service: `sudo systemctl status pifi.service`

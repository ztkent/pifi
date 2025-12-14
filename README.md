# PiFi

Modern WiFi configuration tool for Raspberry Pi.  
Remotely manage IoT projects without physical access to the device.

Works with Bookworm using NetworkManager.  
Tested on Raspberry Pi: 2B, Zero W, Zero 2 W, 4 and 5.

## Key Features

- Web Interface for simple management
- API for programmatic access
- Supports automatically managing offline devices

## Web Interface

A web service that allows you to configure the WiFi settings of your Raspberry Pi.

- Connect to the same network as your device running PiFi
- Navigate directly to `http://<device-ip>:8088`

#### Client Mode
<img width="1000" height="500" alt="Client" src="https://github.com/user-attachments/assets/2de5ba45-c0e4-4135-a0d0-f6c58088268d" />

#### Access Point Mode
<img width="1000" height="500" alt="image" src="https://github.com/user-attachments/assets/67f22add-f276-490b-bd59-f903a335c596" />


## API

You can interact with PiFi programmatically using its API.

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

## Service Setup

Run PiFi as a systemd service for automatic network management on boot.

### Automatic Access Point Mode

When configured as a service, PiFi automatically enables access point mode if no network connection is detected:

- Connect to the `PiFi-AP-<1234>` access point
- Navigate to `http://10.42.0.1:8088`
- Select your WiFi network and connect

### Install as Systemd Service

Create the service file:

```bash
sudo nano /etc/systemd/system/pifi.service
```

Add the following configuration (update paths to match your installation):

```shell
[Unit]
Description=PiFi Service
After=network.target

[Service]
ExecStart=/usr/local/bin/pifi
Environment="PATH=/usr/bin:/usr/sbin"
WorkingDirectory=/usr/local/bin
User=root
Restart=always

[Install]
WantedBy=multi-user.target
```

After modifying the service file, reload and restart:

```bash
sudo systemctl daemon-reload
sudo systemctl restart pifi.service
```

Enable and start the service:

```bash
sudo systemctl daemon-reload
sudo systemctl enable pifi.service
sudo systemctl start pifi.service
sudo systemctl status pifi.service
```

### Configuration Options

PiFi supports the following command-line flags:

| Flag | Default | Description |
|------|---------|-------------|
| `-auto` | `true` | Enable automatic AP mode when no internet connection is detected |
| `-timeout` | `30` | Seconds to wait offline before enabling AP mode |
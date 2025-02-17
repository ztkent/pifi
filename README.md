# PiFi

Modern headless WiFi configuration tool for Raspberry Pi.    
Remotely manage IoT projects without physical access to the device.

Works with Bookworm using NetworkManager.  
Tested on Raspberry Pi: 2B, Zero W, Zero 2 W, and 5. 

## Key Features
- Web Interface for WiFi management
- Access point mode to manage offline devices
- Systemd service for automatic network configuration

## Web Interface

A simple web service that allows you to configure the WiFi settings of your Raspberry Pi.   
- Connect to the same network as your device running PiFi
- Navigate to `http://<device-ip>:8088`  

<img width="720" height="400" alt="image" src="https://github.com/user-attachments/assets/14ca5116-28df-43fd-8203-88c12690f924" />


## Systemd Service

`pifi.service` is a daemon that runs on boot and helps you configure the WiFi settings of your Raspberry Pi.  
If your device is not connected to a network, the service will start in access point mode.

- Connect a client to the access point 
  - The AP should be named `PiFi-AP-<1234>`
- Navigate to `http://10.42.0.1:8088` to view the web interface. 
- View the available networks, and connect your target network.  

### Setup

- Create the new systemd service file:   
`sudo vim /etc/systemd/system/pifi.service`

```shell
[Unit]
Description=PiFi Service
After=network.target

[Service]
ExecStart=<path-to-pifi-binary>
Environment="PATH=/usr/bin:/usr/sbin"
WorkingDirectory=<directory-of-pifi-binary>
User=root
Restart=always

[Install]
WantedBy=multi-user.target
```

- Reload systemd to recognize the new service:   
`sudo systemctl daemon-reload`

- Enable the service to start on boot:   
`sudo systemctl enable pifi.service`

- Start the service immediately:   
`sudo systemctl start pifi.service`

- Check the status of the service:   
`sudo systemctl status pifi.service`

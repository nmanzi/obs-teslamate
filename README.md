# Tesla Location Server for OBS Streaming

A Go application that connects to your Teslamate MQTT broker, retrieves your Tesla's current location and status, and serves it as both a live map (using Leaflet) and a text overlay for OBS Studio.

Perfect for streaming your Perth to Brisbane road trip!

## Features

- 🗺️ **Live Map View**: Real-time location tracking with a rotating car icon based on heading
- 📊 **Status Dashboard**: Battery level, range, speed, and vehicle state
- 📝 **Text Overlay**: Formatted text output for OBS including weather data
- 🌤️ **Weather Integration**: Current weather conditions at your location (using Open-Meteo API)
- 📡 **MQTT Integration**: Connects to your Teslamate MQTT broker
- 🔄 **Auto-updating**: Updates every 2 seconds automatically

## Prerequisites

- Go 1.21 or later
- Teslamate running with MQTT enabled
- MQTT broker (e.g., Mosquitto) running on localhost:1883

## Installation

1. **Install dependencies:**
   ```bash
   go mod download
   ```

2. **Build the application:**
   ```bash
   go build -o tesla-location-server
   ```

3. **Run the server:**
   ```bash
   ./tesla-location-server
   ```

The server will start on port 8080.

## Usage

### Live Map View
Open your browser to:
```
http://localhost:8080
```

This shows:
- Interactive map with your Tesla's location
- Car icon that rotates based on heading
- Info panel with battery, range, speed, and state
- Auto-updates every 2 seconds

### Text Overlay for OBS
Add a Browser Source in OBS pointing to:
```
http://localhost:8080/overlay
```

This provides a formatted text output including:
- GPS coordinates
- Battery percentage and range
- Current speed and state
- Distance from Perth
- Weather conditions (temperature, conditions, humidity, wind)
- Last update timestamp

**OBS Browser Source Settings:**
- Width: 600px
- Height: 400px
- Check "Shutdown source when not visible" for performance
- Refresh browser source every 2-5 seconds using a custom CSS refresh or browser source properties

### JSON API Endpoint
For developers or custom integrations:
```
http://localhost:8080/location
```

Returns JSON with current location data.

## Configuration

### Changing MQTT Settings
Edit `main.go` and modify the MQTT connection settings:

```go
opts := mqtt.NewClientOptions()
opts.AddBroker("tcp://your-mqtt-broker:1883")  // Change broker address
opts.SetUsername("username")                     // Add if authentication required
opts.SetPassword("password")                     // Add if authentication required
```

### Changing Car ID
If your Teslamate car ID is different from "1", update all MQTT topic subscriptions:

```go
"teslamate/cars/YOUR_CAR_ID/latitude"
"teslamate/cars/YOUR_CAR_ID/longitude"
// etc...
```

### Changing Port
Edit the `ListenAndServe` line:

```go
http.ListenAndServe(":8080", nil)  // Change :8080 to your preferred port
```

## Customising the Overlays

### Map Appearance
Edit the Leaflet tile layer in `serveMap()` function. Some alternatives:

```javascript
// Dark mode
L.tileLayer('https://{s}.basemaps.cartocdn.com/dark_all/{z}/{x}/{y}{r}.png', {
    attribution: '© OpenStreetMap contributors'
}).addTo(map);

// Satellite
L.tileLayer('https://server.arcgisonline.com/ArcGIS/rest/services/World_Imagery/MapServer/tile/{z}/{y}/{x}', {
    attribution: 'Esri'
}).addTo(map);
```

### Text Overlay Format
Modify the `serveTextOverlay()` function to customise what information is displayed and how it's formatted.

### Car Icon
Change the car emoji in the marker icon:
```javascript
html: '<div style="font-size: 30px; transform: rotate(' + data.heading + 'deg);">🚙</div>'
```

Try different emojis: 🚗 🚙 🚕 🚐 or use a custom image.

## Troubleshooting

### "Connection lost" errors
- Ensure MQTT broker is running: `sudo systemctl status mosquitto`
- Check broker address is correct (localhost:1883)
- Verify Teslamate is publishing to MQTT

### No location data appearing
- Check Teslamate is running and vehicle is awake
- Verify car ID is correct (check Teslamate web interface)
- Test MQTT topics manually:
  ```bash
  mosquitto_sub -t "teslamate/cars/1/#" -v
  ```

### Weather data shows "Unavailable"
- Check internet connectivity
- Open-Meteo API is free and doesn't require an API key, but ensure outbound HTTPS is allowed

### OBS Browser Source not updating
- Ensure "Shutdown source when not visible" is unchecked if you want continuous updates
- Try adding `&t=` + timestamp to the URL to force refreshes
- Check browser console in OBS for JavaScript errors

## Running as a Service

To run this automatically on boot (Linux):

1. Create `/etc/systemd/system/tesla-location.service`:
```ini
[Unit]
Description=Tesla Location Server
After=network.target mosquitto.service

[Service]
Type=simple
User=your-username
WorkingDirectory=/path/to/tesla-location-server
ExecStart=/path/to/tesla-location-server/tesla-location-server
Restart=always

[Install]
WantedBy=multi-user.target
```

2. Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable tesla-location.service
sudo systemctl start tesla-location.service
```

## Road Trip Tips

For your Perth to Brisbane stream:

1. **Test everything before departure** - make sure all MQTT topics are working
2. **Starlink connectivity** - the server needs internet for weather data
3. **Battery monitoring** - set up alerts if battery drops below certain thresholds
4. **Backup power** - consider UPS for your home server
5. **Remote access** - set up VPN or port forwarding if you need to adjust settings remotely

## Credits

- Maps: OpenStreetMap contributors
- Weather: Open-Meteo API (free, no API key required)
- MQTT: Eclipse Paho
- Mapping: Leaflet.js

## Licence

MIT Licence - feel free to modify and use for your streaming adventure!

---

Good luck with the road trip! 🚗⚡🗺️
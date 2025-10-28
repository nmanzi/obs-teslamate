# Tesla Location Server for OBS Streaming

A Go application that connects to your Teslamate MQTT broker, retrieves your Tesla's current location and status, and serves it as both a live map (using Mapbox) and a text overlay for OBS Studio.

Perfect for IRL streaming during road trips, allowing viewers to see your real-time location, vehicle status, and local weather conditions with real-time configuration management.

## Features

- 🗺️ **Live Map View**: Real-time location tracking with MapBox integration and route visualization
- 🌅 **Dynamic Lighting**: Map automatically adjusts lighting (day/night/dawn/dusk) based on accurate sun calculations (using SunCalc library)
- 📊 **Status Dashboard**: Battery level, range, speed, heading, elevation, and vehicle state
- 📝 **Text Overlay**: Formatted text output for OBS including weather and location data
- 🌤️ **Weather Integration**: Current weather conditions at your location (using Open-Meteo API)
- 🌍 **Location Services**: Neighborhood/city names and local time (using TimeZoneDB API)
- 📡 **MQTT Integration**: Connects to your Teslamate MQTT broker
- 🎛️ **Real-time Configuration**: Admin interface with live config changes (no restart required)
- 🚗 **Route Tracking**: Active route destination, ETA, and arrival battery level
- 🔄 **Auto-updating**: Map updates every 5 seconds, overlay every 10 seconds
- 🛡️ **Secure Admin**: Session-based authentication for configuration changes
- 📱 **Responsive Design**: Works on desktop and mobile devices

## Prerequisites

- Go 1.21 or later
- Teslamate running with MQTT enabled
- MQTT broker (e.g., Mosquitto) accessible from your server
- Mapbox API token (free tier available)
- TimeZoneDB API token (optional, for local time display)

## Installation

1. **Clone the repository:**
   ```bash
   git clone https://github.com/yourusername/obs-teslamate
   cd obs-teslamate
   ```

2. **Install dependencies:**
   ```bash
   go mod download
   ```

3. **Build the application:**
   ```bash
   go build -o tesla-location-server
   ```

4. **Set up environment variables:**
   ```bash
   export MQTT_BROKER="localhost:1883"           # Your MQTT broker address
   export MAPBOX_TOKEN="your_mapbox_token"       # Get from https://mapbox.com
   export TIMEZONEDB_TOKEN="your_timezonedb_token" # Optional: Get from https://timezonedb.com
   export ADMIN_USERNAME="admin"                 # Admin interface username
   export ADMIN_PASSWORD="your_secure_password"  # Admin interface password
   ```

5. **Run the server:**
   ```bash
   ./tesla-location-server
   ```

The server will start on port 8081.

## Usage

### Live Map View
Open your browser to:
```
http://localhost:8081
```

This shows:
- Interactive Mapbox map with your Tesla's location
- Real-time car marker with heading indicator
- Info panel with battery, range, speed, elevation, and state
- Route visualization when navigating to a destination
- ETA and arrival battery information
- Auto-updates every 5 seconds
- **Automatically switches to offline mode** when MapEnabled is disabled

### Text Overlay for OBS
Add a Browser Source in OBS pointing to:
```
http://localhost:8081/overlay
```

This provides a formatted text output including:
- Current location (neighborhood, city, state)
- Distance from home base
- Local time with timezone
- Weather conditions (temperature, description, wind speed)
- Active navigation destination (if any)
- Distance and time to destination
- **Automatically switches to offline mode** when MapEnabled is disabled

**OBS Browser Source Settings:**
- Width: 400px
- Height: 300px
- Check "Shutdown source when not visible" for better performance
- The overlay auto-refreshes every 10 seconds

### Admin Interface
Configure the application in real-time at:
```
http://localhost:8081/admin/login
```

Login with your `ADMIN_USERNAME` and `ADMIN_PASSWORD`, then access the admin panel to control:
- **Map Enabled/Disabled**: Toggle between live map and offline mode
- **Show Route**: Enable/disable route visualization on the map
- **Mapbox Token**: Update API token without restart
- **TimeZoneDB Token**: Update timezone API token without restart

**Changes take effect immediately** - no server restart required!

### JSON API Endpoints

**Location Data:**
```
http://localhost:8081/location
```
Returns real-time Tesla location and status data.

**Local Time:**
```
http://localhost:8081/local-time?lat=LATITUDE&lng=LONGITUDE
```
Returns local time and timezone for given coordinates. Used by the map for dynamic lighting.

**Configuration:**
```
http://localhost:8081/config
```
- GET: Returns current configuration
- POST: Updates configuration (requires JSON body)

**Overlay Data:**
```
http://localhost:8081/overlay-data
```
Returns formatted text content for the overlay.

## Configuration

### Environment Variables

The application uses environment variables for configuration:

```bash
# Required
MQTT_BROKER="your-mqtt-broker:1883"    # MQTT broker address and port
ADMIN_USERNAME="admin"                 # Admin interface username  
ADMIN_PASSWORD="secure_password"       # Admin interface password

# Optional
MAPBOX_TOKEN="your_mapbox_api_token"   # For map functionality
TIMEZONEDB_TOKEN="your_timezone_token" # For local time display
```

### Real-time Configuration

Use the admin interface (`/admin`) to change settings without restarting:

- **Map Enabled**: Toggle between interactive map and offline mode
- **Show Route**: Enable/disable navigation route display
- **Mapbox Token**: Update map API token
- **TimeZoneDB Token**: Update timezone API token

### Changing Car ID

If your Teslamate car ID is different from "1", update the MQTT topic subscriptions in `main.go`:

```go
topics := map[string]byte{
    "teslamate/cars/YOUR_CAR_ID/latitude":             0,
    "teslamate/cars/YOUR_CAR_ID/longitude":            0,
    "teslamate/cars/YOUR_CAR_ID/speed":                0,
    // ... update all topics
}
```

### Changing Server Port

Modify the server port in `main.go`:

```go
log.Fatal(http.ListenAndServe(":8081", nil))  // Change :8081 to your preferred port
```

## Customization

### Map Appearance

The application uses Mapbox with the "standard" style by default. You can customize this in the `templates/root.html` file:

```javascript
var map = new mapboxgl.Map({
    container: 'map',
    style: 'mapbox://styles/mapbox/satellite-v9',  // Change style here
    center: [115.8605, -31.9505],
    zoom: 10
});
```

Available Mapbox styles:
- `mapbox://styles/mapbox/standard` (default)
- `mapbox://styles/mapbox/streets-v12`
- `mapbox://styles/mapbox/outdoors-v12`
- `mapbox://styles/mapbox/light-v11`
- `mapbox://styles/mapbox/dark-v11`
- `mapbox://styles/mapbox/satellite-v9`

### Text Overlay Format

Modify the overlay content in the `serveOverlayData()` function in `main.go`. The format uses standard Go string formatting:

```go
content = fmt.Sprintf(`📍 Location: %s
🎯 Destination: %s
📏 Distance to Destination: %.1f km
🕒 Local Time: %s (%s)
🌡️ Temperature: %.1f°C`, 
    locationName,
    loc.Destination,
    kmToDestination,
    localTime, timezone,
    weather.Temperature)
```

### Map Markers

The car and destination markers are defined as SVG in the JavaScript. You can customize them in `templates/root.html`:

```javascript
// Car marker (red with star)
const markerElement = document.createElement('div');
markerElement.innerHTML = `<svg width="30" height="40">...</svg>`;

// Destination marker (green with circle)
const destElement = document.createElement('div');
destElement.innerHTML = `<svg width="25" height="35">...</svg>`;
```

## Troubleshooting

### "Connection lost" errors
- Ensure MQTT broker is running and accessible
- Check `MQTT_BROKER` environment variable is correct
- Verify Teslamate is publishing to MQTT
- Test MQTT connectivity:
  ```bash
  mosquitto_sub -h your-broker-host -p 1883 -t "teslamate/cars/1/#" -v
  ```

### No location data appearing
- Check Teslamate is running and vehicle is awake
- Verify car ID is correct (check Teslamate web interface)
- Ensure MQTT topics match your car ID
- Check server logs for MQTT connection status

### Map not loading
- Verify `MAPBOX_TOKEN` environment variable is set
- Check Mapbox token is valid and has sufficient quota
- Use admin interface to verify/update token
- Check browser console for JavaScript errors

### Admin interface not accessible
- Verify `ADMIN_USERNAME` and `ADMIN_PASSWORD` are set
- Check you're using the correct URL: `http://localhost:8081/admin/login`
- Clear browser cookies if session issues occur

### Weather data shows "Unavailable"
- Check internet connectivity
- Open-Meteo API is free but requires outbound HTTPS access
- Verify firewall allows HTTPS connections

### Time zone information incorrect
- Set `TIMEZONEDB_TOKEN` environment variable
- Get free API key from https://timezonedb.com
- Falls back to UTC if API unavailable

### OBS Browser Source not updating
- Ensure "Shutdown source when not visible" is unchecked for continuous updates
- Check overlay URL is correct: `http://localhost:8081/overlay`
- Verify server is running and accessible from OBS machine
- Check browser console in OBS for errors

### Real-time config changes not working
- Verify admin session is active
- Check browser console for JavaScript errors
- Config changes should appear within 2 seconds
- Clear browser cache if issues persist

## Running as a Service

To run this automatically on boot (Linux):

1. Create `/etc/systemd/system/tesla-location.service`:
```ini
[Unit]
Description=Tesla Location Server
After=network.target

[Service]
Type=simple
User=your-username
WorkingDirectory=/path/to/obs-teslamate
Environment=MQTT_BROKER=localhost:1883
Environment=MAPBOX_TOKEN=your_mapbox_token
Environment=TIMEZONEDB_TOKEN=your_timezonedb_token
Environment=ADMIN_USERNAME=admin
Environment=ADMIN_PASSWORD=your_secure_password
ExecStart=/path/to/obs-teslamate/tesla-location-server
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

2. Enable and start:
```bash
sudo systemctl daemon-reload
sudo systemctl enable tesla-location.service
sudo systemctl start tesla-location.service
sudo systemctl status tesla-location.service
```

## Docker Deployment

Create a `Dockerfile`:

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o tesla-location-server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/tesla-location-server .
COPY --from=builder /app/templates ./templates
COPY --from=builder /app/public ./public
EXPOSE 8081
CMD ["./tesla-location-server"]
```

Create `docker-compose.yml`:

```yaml
version: '3.8'
services:
  tesla-location:
    build: .
    ports:
      - "8081:8081"
    environment:
      - MQTT_BROKER=mqtt-broker:1883
      - MAPBOX_TOKEN=your_mapbox_token
      - TIMEZONEDB_TOKEN=your_timezonedb_token
      - ADMIN_USERNAME=admin
      - ADMIN_PASSWORD=your_secure_password
    restart: unless-stopped
```

## Road Trip Tips

For optimal IRL streaming experience:

### Before Departure
1. **Test everything** - Verify all MQTT topics, map display, and overlay work correctly
2. **Configure admin access** - Set strong passwords and test the admin interface
3. **Check API quotas** - Ensure Mapbox and TimeZoneDB have sufficient usage limits
4. **Backup configuration** - Document your environment variables

### During the Trip
1. **Internet connectivity** - The server needs internet for:
   - Weather data (Open-Meteo API)
   - Map tiles (Mapbox)
   - Location names (Nominatim)
   - Time zones (TimeZoneDB)
2. **Power management** - Monitor server power consumption
3. **Real-time adjustments** - Use admin interface to toggle features as needed
4. **Bandwidth optimization** - Disable map when on limited data connections

### Emergency Procedures
1. **Offline mode** - Use admin interface to disable map and reduce bandwidth
2. **Manual fallbacks** - Know how to restart services remotely
3. **Backup streaming** - Have alternative overlay sources ready

### Performance Optimization
- **Map disabled**: Saves bandwidth and CPU when not needed
- **Static overlays**: Use offline mode for stable text-only display
- **Update intervals**: Configured for balance between freshness and performance
- **Session management**: Admin sessions auto-expire for security

## API Integration

### Using with Other Services

The JSON endpoints can be integrated with other applications:

```bash
# Get current location
curl http://localhost:8081/location

# Get overlay text
curl http://localhost:8081/overlay-data

# Update configuration
curl -X POST http://localhost:8081/config \
  -H "Content-Type: application/json" \
  -d '{"map_enabled": false, "show_route": true}'
```

### Webhook Integration

You can poll the location endpoint and trigger webhooks for specific events (geofences, low battery, etc.).

## Credits

- **Maps**: Mapbox (API required)
- **Sun Calculations**: SunCalc library for accurate sunrise/sunset times
- **Weather**: Open-Meteo API (free, no API key required)
- **Location Names**: Nominatim/OpenStreetMap (free)
- **Time Zones**: TimeZoneDB API (free tier available)
- **MQTT**: Eclipse Paho Go client
- **Sessions**: Gorilla Sessions
- **Tesla Data**: Teslamate (amazing open-source Tesla logger)

## License

MIT License - feel free to modify and use for your streaming adventures!

## Contributing

Contributions welcome! Please feel free to submit pull requests or open issues for:
- Bug fixes
- Feature enhancements  
- Documentation improvements
- Additional API integrations

---

**Happy streaming and safe travels!** 🚗⚡🗺️📺
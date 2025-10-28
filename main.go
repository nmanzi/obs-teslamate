package main

import (
	"crypto/rand"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/gorilla/sessions"
)

type ActiveRoute struct {
	Destination      string  `json:"destination"`
	EnergyAtArrival  int     `json:"energy_at_arrival"`
	MilesToArrival   float64 `json:"miles_to_arrival"`
	MinutesToArrival float64 `json:"minutes_to_arrival"`
	TrafficDelay     float64 `json:"traffic_minutes_delay"`
	Location         struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	Error string `json:"error"`
}

type Location struct {
	Latitude             float64   `json:"latitude"`
	Longitude            float64   `json:"longitude"`
	Speed                float64   `json:"speed"`
	Heading              float64   `json:"heading"`
	Battery              float64   `json:"battery"`
	Range                float64   `json:"range"`
	State                string    `json:"state"`
	Elevation            float64   `json:"elevation"`
	Destination          string    `json:"destination"`
	DestinationLatitude  float64   `json:"destination_latitude"`
	DestinationLongitude float64   `json:"destination_longitude"`
	MinutesToArrival     float64   `json:"minutes_to_arrival"`
	MilesToArrival       float64   `json:"miles_to_arrival"`
	EnergyAtArrival      int       `json:"energy_at_arrival"`
	UpdatedAt            time.Time `json:"updated_at"`
}

type WeatherData struct {
	Temperature float64 `json:"temperature"`
	Description string  `json:"description"`
	Humidity    int     `json:"humidity"`
	WindSpeed   float64 `json:"wind_speed"`
}

type Config struct {
	ShowRoute       bool   `json:"show_route"`
	MapboxToken     string `json:"mapbox_token"`
	MapEnabled      bool   `json:"map_enabled"`
	OverlayEnabled  bool   `json:"overlay_enabled"`
	TimeZoneDBToken string `json:"timezonedb_token"`
}

var (
	currentLocation Location
	locationMutex   sync.RWMutex
	mqttClient      mqtt.Client
	config          = Config{
		ShowRoute:       true,
		OverlayEnabled:  true,
		MapboxToken:     os.Getenv("MAPBOX_TOKEN"),
		MapEnabled:      true,
		TimeZoneDBToken: os.Getenv("TIMEZONEDB_TOKEN"),
	}
	adminUsername = os.Getenv("ADMIN_USERNAME")
	adminPassword = os.Getenv("ADMIN_PASSWORD")
	mqttBroker    = os.Getenv("MQTT_BROKER")
	sessionStore  *sessions.CookieStore
)

func main() {
	// Initialize session store with a random key
	sessionKey := generateSessionKey()
	sessionStore = sessions.NewCookieStore(sessionKey)
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7, // 7 days
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		SameSite: http.SameSiteStrictMode,
	}

	// Initialize MQTT connection
	opts := mqtt.NewClientOptions()
	opts.AddBroker("tcp://" + mqttBroker)
	opts.SetClientID("tesla-location-server")
	opts.SetDefaultPublishHandler(messagePubHandler)
	opts.OnConnect = connectHandler
	opts.OnConnectionLost = connectLostHandler

	mqttClient = mqtt.NewClient(opts)
	if token := mqttClient.Connect(); token.Wait() && token.Error() != nil {
		log.Fatal(token.Error())
	}

	// Subscribe to Teslamate MQTT topics
	subscribeToTopics()

	// Setup HTTP server
	http.HandleFunc("/{$}", serveRoot)
	http.HandleFunc("/location", serveLocationJSON)
	http.HandleFunc("/local-time", serveLocalTime)
	http.HandleFunc("/overlay", serveOverlay)
	http.HandleFunc("/overlay-data", serveOverlayData)
	http.HandleFunc("/config", serveConfig)
	http.HandleFunc("/admin/login", serveAdminLogin)
	http.HandleFunc("/admin/logout", serveAdminLogout)
	http.HandleFunc("/admin", serveAdmin)
	http.HandleFunc("/admin/config", serveAdminConfig)

	// Serve static files from public directory
	http.Handle("/public/", http.StripPrefix("/public/", http.FileServer(http.Dir("./public/"))))

	log.Println("Starting server on :8081")
	log.Println("Main view (map/offline): http://localhost:8081")
	log.Println("Overlay (live/offline): http://localhost:8081/overlay")
	log.Println("Admin login: http://localhost:8081/admin/login")
	log.Fatal(http.ListenAndServe(":8081", nil))
}

func subscribeToTopics() {
	topics := map[string]byte{
		"teslamate/cars/1/latitude":             0,
		"teslamate/cars/1/longitude":            0,
		"teslamate/cars/1/speed":                0,
		"teslamate/cars/1/heading":              0,
		"teslamate/cars/1/battery_level":        0,
		"teslamate/cars/1/est_battery_range_km": 0,
		"teslamate/cars/1/state":                0,
		"teslamate/cars/1/elevation":            0,
		"teslamate/cars/1/active_route":         0,
	}

	for topic := range topics {
		token := mqttClient.Subscribe(topic, 0, messageHandler)
		token.Wait()
		log.Printf("Subscribed to %s\n", topic)
	}
}

func messageHandler(client mqtt.Client, msg mqtt.Message) {
	locationMutex.Lock()
	defer locationMutex.Unlock()

	topic := msg.Topic()
	payload := string(msg.Payload())

	switch topic {
	case "teslamate/cars/1/latitude":
		if lat, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Latitude = lat
			currentLocation.UpdatedAt = time.Now()
		}
	case "teslamate/cars/1/longitude":
		if lon, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Longitude = lon
			currentLocation.UpdatedAt = time.Now()
		}
	case "teslamate/cars/1/speed":
		if speed, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Speed = speed
		}
	case "teslamate/cars/1/heading":
		if heading, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Heading = heading
		}
	case "teslamate/cars/1/battery_level":
		if battery, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Battery = battery
		}
	case "teslamate/cars/1/est_battery_range_km":
		if rng, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Range = rng
		}
	case "teslamate/cars/1/state":
		currentLocation.State = payload
	case "teslamate/cars/1/elevation":
		if elevation, err := strconv.ParseFloat(payload, 64); err == nil {
			currentLocation.Elevation = elevation
		}
	case "teslamate/cars/1/active_route":
		var route ActiveRoute
		if err := json.Unmarshal([]byte(payload), &route); err == nil {
			if route.Error == "" || route.Error == "null" {
				// Active route available
				currentLocation.Destination = route.Destination
				currentLocation.DestinationLatitude = route.Location.Latitude
				currentLocation.DestinationLongitude = route.Location.Longitude
				currentLocation.MinutesToArrival = route.MinutesToArrival
				currentLocation.MilesToArrival = route.MilesToArrival
				currentLocation.EnergyAtArrival = route.EnergyAtArrival
			} else {
				// No active route
				currentLocation.Destination = ""
				currentLocation.DestinationLatitude = 0
				currentLocation.DestinationLongitude = 0
				currentLocation.MinutesToArrival = 0
				currentLocation.MilesToArrival = 0
				currentLocation.EnergyAtArrival = 0
			}
		}
	}
}

func messagePubHandler(client mqtt.Client, msg mqtt.Message) {
	// Default handler
}

func connectHandler(client mqtt.Client) {
	log.Println("Connected to MQTT broker")
}

func connectLostHandler(client mqtt.Client, err error) {
	log.Printf("Connection lost: %v\n", err)
}

func generateSessionKey() []byte {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		log.Fatal("Failed to generate session key:", err)
	}
	return key
}

func serveRoot(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("templates/root.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, config)
}

func serveLocationJSON(w http.ResponseWriter, r *http.Request) {
	if config.MapEnabled {
		locationMutex.RLock()
		defer locationMutex.RUnlock()

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(currentLocation)
	} else {
		http.Error(w, "Map is disabled in configuration.", http.StatusForbidden)
	}
}

func serveLocalTime(w http.ResponseWriter, r *http.Request) {
	// Parse latitude and longitude from query parameters
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")

	if latStr == "" || lngStr == "" {
		http.Error(w, "Missing lat or lng parameters", http.StatusBadRequest)
		return
	}

	lat, err := strconv.ParseFloat(latStr, 64)
	if err != nil {
		http.Error(w, "Invalid latitude parameter", http.StatusBadRequest)
		return
	}

	lng, err := strconv.ParseFloat(lngStr, 64)
	if err != nil {
		http.Error(w, "Invalid longitude parameter", http.StatusBadRequest)
		return
	}

	// Get local time for the given coordinates
	localTime, timezone := getLocalTime(lat, lng)

	// Return as JSON
	response := map[string]string{
		"time":     localTime,
		"timezone": timezone,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func serveOverlay(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("templates/overlay.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, nil)
}

type OverlayData struct {
	Content string `json:"content"`
}

func serveOverlayData(w http.ResponseWriter, r *http.Request) {
	var overlayData OverlayData

	// Build overlay content if overlay is enabled
	if config.OverlayEnabled {
		locationMutex.RLock()
		loc := currentLocation
		locationMutex.RUnlock()

		// Get location name (neighborhood/city)
		locationName := getLocationName(loc.Latitude, loc.Longitude)

		// Get timezone and local time
		localTime, timezone := getLocalTime(loc.Latitude, loc.Longitude)

		// Get weather data
		weather := getWeather(loc.Latitude, loc.Longitude)

		// Calculate distance from Baldivis, WA (approximate)
		distanceFromPerth := calculateDistance(-32.2833, 115.8420, loc.Latitude, loc.Longitude)

		// Build content with optional destination info
		var content string
		if loc.Destination != "" {
			// Convert miles to kilometers for distance to destination
			kmToDestination := loc.MilesToArrival * 1.60934
			content = fmt.Sprintf(`📍 Location: %s
🎯 Destination: %s
📏 Distance to Destination: %.1f km
📏 Distance from Home: %.0f km

🕒 Local Time: %s (%s)
🌡️ Temperature: %.1f°C
🌤️ Conditions: %s
💨 Wind: %.1f km/h`,
				locationName,
				loc.Destination,
				kmToDestination,
				distanceFromPerth,
				localTime, timezone,
				weather.Temperature,
				weather.Description,
				weather.WindSpeed)
		} else {
			content = fmt.Sprintf(`📍 Location: %s
📏 Distance from Home: %.0f km

🕒 Local Time: %s (%s)
🌡️ Temperature: %.1f°C
🌤️ Conditions: %s
💨 Wind: %.1f km/h`,
				locationName,
				distanceFromPerth,
				localTime, timezone,
				weather.Temperature,
				weather.Description,
				weather.WindSpeed)
		}

		overlayData = OverlayData{Content: content}
	} else {
		overlayData = OverlayData{Content: "Overlay is disabled in configuration."}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(overlayData)
}

func serveConfig(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "GET":
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func requireAuth(w http.ResponseWriter, r *http.Request) bool {
	session, err := sessionStore.Get(r, "admin-session")
	if err != nil {
		log.Printf("Session error: %v", err)
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return false
	}

	if auth, ok := session.Values["authenticated"].(bool); !ok || !auth {
		http.Redirect(w, r, "/admin/login", http.StatusFound)
		return false
	}

	// Check session timeout (24 hours)
	if loginTime, ok := session.Values["login_time"].(time.Time); ok {
		if time.Since(loginTime) > 24*time.Hour {
			// Session expired
			session.Values["authenticated"] = false
			session.Save(r, w)
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return false
		}
	}

	return true
}

func serveAdmin(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}

	// Get session info for template
	session, _ := sessionStore.Get(r, "admin-session")
	username := "admin"
	if u, ok := session.Values["username"].(string); ok {
		username = u
	}

	data := map[string]interface{}{
		"Username": username,
	}

	w.Header().Set("Content-Type", "text/html")
	t, err := template.ParseFiles("templates/admin.html")
	if err != nil {
		http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
		return
	}
	t.Execute(w, data)
}

func serveAdminConfig(w http.ResponseWriter, r *http.Request) {
	if !requireAuth(w, r) {
		return
	}

	switch r.Method {
	case "POST":
		var newConfig Config
		if err := json.NewDecoder(r.Body).Decode(&newConfig); err != nil {
			http.Error(w, "Invalid JSON", http.StatusBadRequest)
			return
		}
		config = newConfig
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(config)
	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func serveAdminLogin(w http.ResponseWriter, r *http.Request) {
	if adminUsername == "" || adminPassword == "" {
		http.Error(w, "Admin authentication not configured. Set ADMIN_USERNAME and ADMIN_PASSWORD environment variables.", http.StatusInternalServerError)
		return
	}

	switch r.Method {
	case "GET":
		// Show login form
		w.Header().Set("Content-Type", "text/html")
		t, err := template.ParseFiles("templates/login.html")
		if err != nil {
			http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
			return
		}
		t.Execute(w, nil)

	case "POST":
		// Process login
		username := r.FormValue("username")
		password := r.FormValue("password")

		if username == adminUsername && password == adminPassword {
			session, err := sessionStore.Get(r, "admin-session")
			if err != nil {
				log.Printf("Session error during login: %v", err)
			}

			session.Values["authenticated"] = true
			session.Values["username"] = username
			session.Values["login_time"] = time.Now().String()

			if err := session.Save(r, w); err != nil {
				http.Error(w, "Failed to save session: "+err.Error(), http.StatusInternalServerError)
				return
			}

			http.Redirect(w, r, "/admin", http.StatusFound)
		} else {
			// Login failed, show form again with error
			w.Header().Set("Content-Type", "text/html")
			t, err := template.ParseFiles("templates/login.html")
			if err != nil {
				http.Error(w, "Template error: "+err.Error(), http.StatusInternalServerError)
				return
			}
			data := map[string]interface{}{
				"Error": "Invalid username or password",
			}
			t.Execute(w, data)
		}

	default:
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
	}
}

func serveAdminLogout(w http.ResponseWriter, r *http.Request) {
	session, err := sessionStore.Get(r, "admin-session")
	if err != nil {
		log.Printf("Session error during logout: %v", err)
	}

	// Clear the session
	session.Values["authenticated"] = false
	delete(session.Values, "username")
	delete(session.Values, "login_time")
	session.Options.MaxAge = -1 // Delete the cookie

	if err := session.Save(r, w); err != nil {
		log.Printf("Failed to clear session: %v", err)
	}

	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func getLocalTime(lat, lon float64) (string, string) {
	// Using TimeZoneDB API with provided API key
	apiKey := config.TimeZoneDBToken
	url := fmt.Sprintf("http://api.timezonedb.com/v2.1/get-time-zone?key=%s&format=json&by=position&lat=%.6f&lng=%.6f", apiKey, lat, lon)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching timezone: %v", err)
		// Fallback to UTC
		now := time.Now().UTC()
		return now.Format("15:04:05"), "UTC"
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		// Fallback to UTC
		now := time.Now().UTC()
		return now.Format("15:04:05"), "UTC"
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		// Fallback to UTC
		now := time.Now().UTC()
		return now.Format("15:04:05"), "UTC"
	}

	// Check if the API call was successful
	if status, ok := result["status"].(string); !ok || status != "OK" {
		log.Printf("TimeZoneDB API error: %v", result["message"])
		// Fallback to UTC
		now := time.Now().UTC()
		return now.Format("15:04:05"), "UTC"
	}

	// Extract timezone info from TimeZoneDB response
	timezoneName := "UTC"
	if tz, ok := result["zoneName"].(string); ok && tz != "" {
		timezoneName = tz
	}

	// Get formatted time directly from the API response if available
	if formatted, ok := result["formatted"].(string); ok && formatted != "" {
		// Parse the formatted time from TimeZoneDB (format: "2023-10-25 14:30:15")
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", formatted); err == nil {
			// Format timezone display name (remove long path, show just the key part)
			timezoneDisplay := timezoneName
			if parts := strings.Split(timezoneName, "/"); len(parts) > 1 {
				timezoneDisplay = parts[len(parts)-1]
				// Replace underscores with spaces for readability
				timezoneDisplay = strings.ReplaceAll(timezoneDisplay, "_", " ")
			}

			return parsedTime.Format("15:04:05"), timezoneDisplay
		}
	}

	// Fallback: use Go's timezone handling
	loc, err := time.LoadLocation(timezoneName)
	if err != nil {
		// If we can't load the timezone, fall back to UTC
		now := time.Now().UTC()
		return now.Format("15:04:05"), "UTC"
	}

	now := time.Now().In(loc)

	// Format timezone display name (remove long path, show just the key part)
	timezoneDisplay := timezoneName
	if parts := strings.Split(timezoneName, "/"); len(parts) > 1 {
		timezoneDisplay = parts[len(parts)-1]
		// Replace underscores with spaces for readability
		timezoneDisplay = strings.ReplaceAll(timezoneDisplay, "_", " ")
	}

	return now.Format("15:04:05"), timezoneDisplay
}

func getLocationName(lat, lon float64) string {
	// Using Nominatim API (OpenStreetMap's free geocoding service)
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?format=json&lat=%.6f&lon=%.6f&zoom=14&addressdetails=1", lat, lon)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching location name: %v", err)
		return fmt.Sprintf("%.4f°, %.4f°", lat, lon)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("%.4f°, %.4f°", lat, lon)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Sprintf("%.4f°, %.4f°", lat, lon)
	}

	// Try to extract meaningful location information
	if address, ok := result["address"].(map[string]interface{}); ok {
		// Priority order: suburb/neighbourhood -> city -> town -> village -> county -> state
		locationParts := []string{}

		if suburb, ok := address["suburb"].(string); ok && suburb != "" {
			locationParts = append(locationParts, suburb)
		} else if neighbourhood, ok := address["neighbourhood"].(string); ok && neighbourhood != "" {
			locationParts = append(locationParts, neighbourhood)
		}

		if city, ok := address["city"].(string); ok && city != "" {
			locationParts = append(locationParts, city)
		} else if town, ok := address["town"].(string); ok && town != "" {
			locationParts = append(locationParts, town)
		} else if village, ok := address["village"].(string); ok && village != "" {
			locationParts = append(locationParts, village)
		}

		if state, ok := address["state"].(string); ok && state != "" {
			locationParts = append(locationParts, state)
		}

		if len(locationParts) > 0 {
			if len(locationParts) == 1 {
				return locationParts[0]
			}
			return fmt.Sprintf("%s, %s", locationParts[0], locationParts[len(locationParts)-1])
		}
	}

	// Fallback to display_name if available
	if displayName, ok := result["display_name"].(string); ok && displayName != "" {
		// Take first part before first comma (usually the most specific location)
		parts := strings.Split(displayName, ",")
		if len(parts) > 0 {
			return strings.TrimSpace(parts[0])
		}
	}

	// Final fallback to coordinates
	return fmt.Sprintf("%.4f°, %.4f°", lat, lon)
}

func getWeather(lat, lon float64) WeatherData {
	// Using Open-Meteo API (free, no API key required)
	url := fmt.Sprintf("https://api.open-meteo.com/v1/forecast?latitude=%.4f&longitude=%.4f&current=temperature_2m,relative_humidity_2m,weather_code,wind_speed_10m", lat, lon)

	resp, err := http.Get(url)
	if err != nil {
		log.Printf("Error fetching weather: %v", err)
		return WeatherData{Description: "Unavailable"}
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return WeatherData{Description: "Unavailable"}
	}

	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		return WeatherData{Description: "Unavailable"}
	}

	current, ok := result["current"].(map[string]interface{})
	if !ok {
		return WeatherData{Description: "Unavailable"}
	}

	temp := 0.0
	if t, ok := current["temperature_2m"].(float64); ok {
		temp = t
	}

	humidity := 0
	if h, ok := current["relative_humidity_2m"].(float64); ok {
		humidity = int(h)
	}

	windSpeed := 0.0
	if w, ok := current["wind_speed_10m"].(float64); ok {
		windSpeed = w
	}

	weatherCode := 0.0
	if wc, ok := current["weather_code"].(float64); ok {
		weatherCode = wc
	}

	return WeatherData{
		Temperature: temp,
		Description: weatherCodeToDescription(int(weatherCode)),
		Humidity:    humidity,
		WindSpeed:   windSpeed,
	}
}

func weatherCodeToDescription(code int) string {
	// WMO Weather interpretation codes
	descriptions := map[int]string{
		0:  "Clear sky",
		1:  "Mainly clear",
		2:  "Partly cloudy",
		3:  "Overcast",
		45: "Foggy",
		48: "Depositing rime fog",
		51: "Light drizzle",
		53: "Moderate drizzle",
		55: "Dense drizzle",
		61: "Slight rain",
		63: "Moderate rain",
		65: "Heavy rain",
		71: "Slight snow",
		73: "Moderate snow",
		75: "Heavy snow",
		77: "Snow grains",
		80: "Slight rain showers",
		81: "Moderate rain showers",
		82: "Violent rain showers",
		85: "Slight snow showers",
		86: "Heavy snow showers",
		95: "Thunderstorm",
		96: "Thunderstorm with slight hail",
		99: "Thunderstorm with heavy hail",
	}

	if desc, ok := descriptions[code]; ok {
		return desc
	}
	return "Unknown"
}

func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	// Haversine formula for distance calculation
	const R = 6371 // Earth's radius in km

	dLat := (lat2 - lat1) * 3.14159265359 / 180
	dLon := (lon2 - lon1) * 3.14159265359 / 180

	a := 0.5 - 0.5*cosApprox(dLat) + cosApprox(lat1*3.14159265359/180)*cosApprox(lat2*3.14159265359/180)*(1-cosApprox(dLon))/2

	return R * 2 * asinApprox(sqrtApprox(a))
}

func cosApprox(x float64) float64 {
	// Simple cosine approximation
	x = x - float64(int(x/(2*3.14159265359)))*(2*3.14159265359)
	if x < 0 {
		x = -x
	}
	if x > 3.14159265359 {
		return -cosApprox(x - 3.14159265359)
	}
	x2 := x * x
	return 1 - x2/2 + x2*x2/24
}

func asinApprox(x float64) float64 {
	return x + x*x*x/6 + 3*x*x*x*x*x/40
}

func sqrtApprox(x float64) float64 {
	if x == 0 {
		return 0
	}
	z := x
	for i := 0; i < 10; i++ {
		z = (z + x/z) / 2
	}
	return z
}

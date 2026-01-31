package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"gopkg.in/yaml.v3"
)

// Config structures
type Direction struct {
	Label  string `yaml:"label" json:"label"`
	StopID string `yaml:"stop_id" json:"stop_id"`
}

type Stop struct {
	Name       string      `yaml:"name" json:"name"`
	Line       string      `yaml:"line" json:"line"`
	Agency     string      `yaml:"agency" json:"agency"`
	Directions []Direction `yaml:"directions" json:"directions"`
}

type Config struct {
	APIKey               string `yaml:"api_key"`
	RefreshInterval      int    `yaml:"refresh_interval"`
	CacheRefreshInterval int    `yaml:"cache_refresh_interval"`
	Port                 int    `yaml:"port"`
	Stops                []Stop `yaml:"stops"`
}

// API response structures
type Arrival struct {
	ArrivalTime string `json:"arrival_time"`
	Minutes     int    `json:"minutes"`
	Destination string `json:"destination"`
	LineType    string `json:"line_type,omitempty"`
}

type DirectionArrivals struct {
	Label          string    `json:"label"`
	StopID         string    `json:"stop_id"`
	Arrivals       []Arrival `json:"arrivals"`
	Error          string    `json:"error,omitempty"`
	QualityWarning string    `json:"quality_warning,omitempty"`
	QualityLevel   string    `json:"quality_level,omitempty"`
}

type StopArrivals struct {
	Name       string              `json:"name"`
	Line       string              `json:"line"`
	Directions []DirectionArrivals `json:"directions"`
}

type ArrivalsResponse struct {
	Stops       []StopArrivals `json:"stops"`
	LastUpdated string         `json:"last_updated"`
}

type ConfigResponse struct {
	Stops           []Stop `json:"stops"`
	RefreshInterval int    `json:"refresh_interval"`
}

// 511.org API response structures
type MonitoredCall struct {
	ExpectedArrivalTime   string `json:"ExpectedArrivalTime"`
	ExpectedDepartureTime string `json:"ExpectedDepartureTime"`
}

type MonitoredVehicleJourney struct {
	LineRef         string        `json:"LineRef"`
	DestinationName string        `json:"DestinationName"`
	MonitoredCall   MonitoredCall `json:"MonitoredCall"`
}

type MonitoredStopVisit struct {
	MonitoredVehicleJourney MonitoredVehicleJourney `json:"MonitoredVehicleJourney"`
}

type StopMonitoringDelivery struct {
	MonitoredStopVisit []MonitoredStopVisit `json:"MonitoredStopVisit"`
}

type ServiceDelivery struct {
	StopMonitoringDelivery StopMonitoringDelivery `json:"StopMonitoringDelivery"`
}

type APIResponse struct {
	ServiceDelivery ServiceDelivery `json:"ServiceDelivery"`
}

var config Config

// Shared HTTP client with connection pooling
var httpClient = &http.Client{
	Timeout: 15 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 5,
		IdleConnTimeout:     30 * time.Second,
	},
}

// Cache for arrivals data
type ArrivalsCache struct {
	mu          sync.RWMutex
	data        ArrivalsResponse
	lastFetched time.Time
}

var cache = &ArrivalsCache{}

func loadConfig() error {
	configPath := "config.yaml"
	if envPath := os.Getenv("CONFIG_PATH"); envPath != "" {
		configPath = envPath
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	if config.APIKey == "" {
		return fmt.Errorf("api_key is required in config")
	}

	if len(config.Stops) == 0 {
		return fmt.Errorf("at least one stop must be configured")
	}

	if config.RefreshInterval == 0 {
		config.RefreshInterval = 30
	}

	if config.Port == 0 {
		config.Port = 8080
	}

	return nil
}

func fetchStopArrivals(agency, stopID string) ([]Arrival, error) {
	if agency == "" {
		agency = "SF"
	}
	url := fmt.Sprintf(
		"https://api.511.org/transit/StopMonitoring?api_key=%s&agency=%s&stopCode=%s&format=json",
		config.APIKey, agency, stopID,
	)

	resp, err := httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body[:min(len(body), 100)]))
	}

	// Strip UTF-8 BOM if present
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	arrivals := make([]Arrival, 0)

	for _, visit := range apiResp.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit {
		// Use arrival time, or departure time if arrival is not available
		timeStr := visit.MonitoredVehicleJourney.MonitoredCall.ExpectedArrivalTime
		if timeStr == "" {
			timeStr = visit.MonitoredVehicleJourney.MonitoredCall.ExpectedDepartureTime
		}
		if timeStr == "" {
			continue
		}

		// Validate the timestamp can be parsed
		_, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}

		arrivals = append(arrivals, Arrival{
			ArrivalTime: timeStr,
			Destination: visit.MonitoredVehicleJourney.DestinationName,
			LineType:    visit.MonitoredVehicleJourney.LineRef,
		})
	}

	return arrivals, nil
}

// detectQualityIssues analyzes arrivals and returns warning message and level
func detectQualityIssues(arrivals []Arrival, now time.Time) (string, string) {
	if len(arrivals) == 0 {
		return "", "good"
	}

	// Parse arrival times
	times := make([]time.Time, 0, len(arrivals))
	for _, arr := range arrivals {
		t, err := time.Parse(time.RFC3339, arr.ArrivalTime)
		if err != nil {
			continue
		}
		times = append(times, t)
	}

	if len(times) == 0 {
		return "", "good"
	}

	// Check 1: Large gaps (>40 mins)
	for i := 1; i < len(times); i++ {
		gap := times[i].Sub(times[i-1]).Minutes()
		if gap > 40 {
			return "Incomplete data - large gap in arrivals", "warning"
		}
	}

	// Check 2: Far future first arrival during normal hours
	firstMinutes := times[0].Sub(now).Minutes()
	hour := now.Hour()
	isNormalHours := hour >= 6 && hour < 22

	if isNormalHours && firstMinutes > 90 {
		return "Next arrival unusually far away", "warning"
	}

	// Check 3: Sparse data during peak hours
	isPeakHours := (hour >= 7 && hour <= 9) || (hour >= 16 && hour <= 19)
	if isPeakHours && len(times) == 1 && firstMinutes < 90 {
		return "Limited schedule data available", "warning"
	}

	return "", "good"
}

// refreshCache fetches all stops sequentially with delays to avoid rate limiting
func refreshCache() {
	log.Println("Refreshing arrivals cache...")

	response := ArrivalsResponse{
		Stops:       make([]StopArrivals, len(config.Stops)),
		LastUpdated: time.Now().Format("3:04:05 PM"),
	}

	for i, stop := range config.Stops {
		response.Stops[i] = StopArrivals{
			Name:       stop.Name,
			Line:       stop.Line,
			Directions: make([]DirectionArrivals, len(stop.Directions)),
		}

		for j, dir := range stop.Directions {
			response.Stops[i].Directions[j] = DirectionArrivals{
				Label:    dir.Label,
				StopID:   dir.StopID,
				Arrivals: []Arrival{},
			}

			arrivals, err := fetchStopArrivals(stop.Agency, dir.StopID)
			if err != nil {
				response.Stops[i].Directions[j].Error = "Unable to fetch"
				log.Printf("Error fetching %s (stop %s): %v", dir.Label, dir.StopID, err)
			} else {
				response.Stops[i].Directions[j].Arrivals = arrivals
				log.Printf("Fetched %s: %d arrivals", dir.Label, len(arrivals))
			}

			// Wait 1.5 seconds between API calls to avoid rate limiting
			// 60 requests/hour = 1 per minute allowed, but we batch them
			time.Sleep(1500 * time.Millisecond)
		}
	}

	// Update cache
	cache.mu.Lock()
	cache.data = response
	cache.lastFetched = time.Now()
	cache.mu.Unlock()

	log.Println("Cache refresh complete")
}

// startCacheRefresher runs the cache refresh in the background
func startCacheRefresher() {
	// Initial fetch
	refreshCache()

	// Count total directions to calculate refresh interval
	totalDirections := 0
	for _, stop := range config.Stops {
		totalDirections += len(stop.Directions)
	}

	// Use configured interval or default to 240 seconds (4 minutes)
	// With 60 req/hour limit: 60 / totalDirections = max refreshes per hour
	// Example: 4 directions = 15 refreshes/hour = 4 minute intervals minimum
	refreshInterval := time.Duration(config.CacheRefreshInterval) * time.Second
	if refreshInterval == 0 {
		refreshInterval = 4 * time.Minute
	}

	log.Printf("Cache will refresh every %v (%d directions)", refreshInterval, totalDirections)

	ticker := time.NewTicker(refreshInterval)
	go func() {
		for range ticker.C {
			refreshCache()
		}
	}()
}

func handleArrivals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	cache.mu.RLock()
	cachedData := cache.data
	cache.mu.RUnlock()

	// If cache is empty, return empty response
	if len(cachedData.Stops) == 0 {
		response := ArrivalsResponse{
			Stops:       make([]StopArrivals, 0),
			LastUpdated: "Loading...",
		}
		json.NewEncoder(w).Encode(response)
		return
	}

	// Create a fresh response with recalculated minutes
	response := ArrivalsResponse{
		Stops:       make([]StopArrivals, len(cachedData.Stops)),
		LastUpdated: time.Now().Format("3:04:05 PM"),
	}

	now := time.Now()

	for i, stop := range cachedData.Stops {
		response.Stops[i] = StopArrivals{
			Name:       stop.Name,
			Line:       stop.Line,
			Directions: make([]DirectionArrivals, len(stop.Directions)),
		}

		for j, dir := range stop.Directions {
			response.Stops[i].Directions[j] = DirectionArrivals{
				Label:    dir.Label,
				StopID:   dir.StopID,
				Arrivals: make([]Arrival, 0),
				Error:    dir.Error,
			}

			// Skip if there was an error fetching this direction
			if dir.Error != "" {
				continue
			}

			// Recalculate minutes for each arrival
			validArrivals := make([]Arrival, 0)
			for _, arrival := range dir.Arrivals {
				arrivalTime, err := time.Parse(time.RFC3339, arrival.ArrivalTime)
				if err != nil {
					continue
				}

				minutes := int(arrivalTime.Sub(now).Minutes())
				if minutes < 0 {
					continue // Skip arrivals in the past
				}

				validArrivals = append(validArrivals, Arrival{
					ArrivalTime: arrival.ArrivalTime,
					Minutes:     minutes,
					Destination: arrival.Destination,
					LineType:    arrival.LineType,
				})
			}

			// Limit to 3 upcoming arrivals
			if len(validArrivals) > 3 {
				validArrivals = validArrivals[:3]
			}

			// Detect quality issues
			warningMsg, qualityLevel := detectQualityIssues(validArrivals, now)

			response.Stops[i].Directions[j].Arrivals = validArrivals
			response.Stops[i].Directions[j].QualityWarning = warningMsg
			response.Stops[i].Directions[j].QualityLevel = qualityLevel
		}
	}

	json.NewEncoder(w).Encode(response)
}

func handleConfig(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ConfigResponse{
		Stops:           config.Stops,
		RefreshInterval: config.RefreshInterval,
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := loadConfig(); err != nil {
		log.Fatalf("Configuration error: %v", err)
	}

	log.Printf("Loaded config with %d stops", len(config.Stops))

	// Start background cache refresher
	startCacheRefresher()

	// API routes
	http.HandleFunc("/api/arrivals", handleArrivals)
	http.HandleFunc("/api/config", handleConfig)
	http.HandleFunc("/health", handleHealth)

	// Static files
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)

	addr := fmt.Sprintf(":%d", config.Port)
	log.Printf("Server starting on http://localhost%s", addr)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatalf("Server failed: %v", err)
	}
}

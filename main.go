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
	APIKey          string `yaml:"api_key"`
	RefreshInterval int    `yaml:"refresh_interval"`
	Port            int    `yaml:"port"`
	Stops           []Stop `yaml:"stops"`
}

// API response structures
type Arrival struct {
	Minutes     int    `json:"minutes"`
	Destination string `json:"destination"`
	LineType    string `json:"line_type,omitempty"`
}

type DirectionArrivals struct {
	Label    string    `json:"label"`
	StopID   string    `json:"stop_id"`
	Arrivals []Arrival `json:"arrivals"`
	Error    string    `json:"error,omitempty"`
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

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Strip UTF-8 BOM if present
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	arrivals := make([]Arrival, 0)
	now := time.Now()

	for _, visit := range apiResp.ServiceDelivery.StopMonitoringDelivery.MonitoredStopVisit {
		// Use arrival time, or departure time if arrival is not available
		timeStr := visit.MonitoredVehicleJourney.MonitoredCall.ExpectedArrivalTime
		if timeStr == "" {
			timeStr = visit.MonitoredVehicleJourney.MonitoredCall.ExpectedDepartureTime
		}
		if timeStr == "" {
			continue
		}

		departTime, err := time.Parse(time.RFC3339, timeStr)
		if err != nil {
			continue
		}

		minutes := int(departTime.Sub(now).Minutes())
		if minutes < 0 {
			minutes = 0
		}

		arrivals = append(arrivals, Arrival{
			Minutes:     minutes,
			Destination: visit.MonitoredVehicleJourney.DestinationName,
			LineType:    visit.MonitoredVehicleJourney.LineRef,
		})
	}

	return arrivals, nil
}

func handleArrivals(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	response := ArrivalsResponse{
		Stops:       make([]StopArrivals, len(config.Stops)),
		LastUpdated: time.Now().Format("3:04:05 PM"),
	}

	var wg sync.WaitGroup

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

			wg.Add(1)
			go func(i, j int, agency, stopID string) {
				defer wg.Done()
				arrivals, err := fetchStopArrivals(agency, stopID)
				if err != nil {
					response.Stops[i].Directions[j].Error = "Unable to fetch"
					log.Printf("Error fetching stop %s: %v", stopID, err)
					return
				}
				// Limit to 3 arrivals
				if len(arrivals) > 3 {
					arrivals = arrivals[:3]
				}
				response.Stops[i].Directions[j].Arrivals = arrivals
			}(i, j, stop.Agency, dir.StopID)
		}
	}

	wg.Wait()

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

# Muni Quick Tracker

A fast, glanceable web app showing real-time SF Muni arrivals for your favorite stops.

## Features

- Real-time arrivals from 511.org API
- Two preconfigured stops with both directions
- Auto-refresh every 30 seconds (pauses when tab hidden)
- Mobile-first responsive design
- Single Docker container deployment

## Quick Start

### 1. Get a 511.org API Key

Get your free API key at [511.org/open-data](https://511.org/open-data)

### 2. Configure

```bash
cp config.example.yaml config.yaml
# Edit config.yaml and add your API key
```

### 3. Run with Docker

```bash
docker-compose up -d
```

Open http://localhost:8080

### Run Locally (Development)

```bash
# Install Go 1.21+
go run main.go
```

## Configuration

Edit `config.yaml`:

```yaml
api_key: "YOUR_511_API_KEY"
refresh_interval: 30  # seconds
port: 8080

stops:
  - name: "T Line"
    line: "T Third"
    directions:
      - label: "Chinatown"
        stop_id: "17166"
      - label: "Sunnydale"
        stop_id: "17397"

  - name: "N Line"
    line: "N Judah"
    directions:
      - label: "Ocean Beach"
        stop_id: "15240"
      - label: "Downtown"
        stop_id: "15239"
```

### Finding Stop IDs

Use the 511.org API to find stop IDs:

```bash
curl "https://api.511.org/transit/stops?api_key=YOUR_KEY&operator_id=SF&format=json"
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI |
| `GET /api/arrivals` | Real-time arrivals JSON |
| `GET /api/config` | Current configuration (no API key) |
| `GET /health` | Health check |

## License

MIT

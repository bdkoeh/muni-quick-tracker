# Muni Quick Tracker

A fast, glanceable web app showing real-time SF Muni and Caltrain arrivals.

## Features

- Real-time arrivals from 511.org API
- Supports SF Muni and Caltrain
- Caltrain shows train types (Express, Limited, Local)
- Server-side caching to stay within API rate limits
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
refresh_interval: 30  # frontend refresh (seconds)
port: 8080

stops:
  - name: "Powell Station"
    line: "F Market"
    agency: "SF"
    directions:
      - label: "Fisherman's Wharf"
        stop_id: "15731"
      - label: "Castro"
        stop_id: "15730"

  - name: "Embarcadero"
    line: "N Judah"
    agency: "SF"
    directions:
      - label: "Ocean Beach"
        stop_id: "16994"

  - name: "Caltrain"
    line: "Caltrain"
    agency: "CT"
    directions:
      - label: "Southbound"
        stop_id: "70012"
```

### Supported Agencies

| Agency | Code | Description |
|--------|------|-------------|
| SF Muni | `SF` | San Francisco Municipal Railway |
| Caltrain | `CT` | Peninsula commuter rail |

### Finding Stop IDs

Use the 511.org API to find stop IDs:

```bash
# SF Muni stops
curl "https://api.511.org/transit/stops?api_key=YOUR_KEY&operator_id=SF&format=json"

# Caltrain stops
curl "https://api.511.org/transit/stops?api_key=YOUR_KEY&operator_id=CT&format=json"
```

## Rate Limits

The 511.org API allows **60 requests per hour**. The server caches arrivals and refreshes every 5 minutes to stay well under this limit.

- 4 directions × 12 refreshes/hour = 48 requests/hour
- Frontend refreshes from cache (no API calls)

## Deployment (Unraid/Docker)

Export the image:
```bash
docker save muni_quick_tracker-muni-tracker:latest | gzip > muni-tracker.tar.gz
```

On your server:
```bash
# Load image
docker load < muni-tracker.tar.gz

# Run
docker run -d \
  --name muni-tracker \
  -p 8080:8080 \
  -v /path/to/config.yaml:/app/config.yaml:ro \
  --restart unless-stopped \
  muni_quick_tracker-muni-tracker:latest
```

## API Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /` | Web UI |
| `GET /api/arrivals` | Cached arrivals JSON |
| `GET /api/config` | Current configuration (no API key) |
| `GET /health` | Health check |

## License

MIT

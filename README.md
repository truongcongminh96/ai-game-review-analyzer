# AI Game Review Analyzer

A backend service that analyzes player reviews from Steam using a local LLM (Ollama).  
The system extracts sentiment, praised features, common issues, gameplay topics, and generates a summary report.

The current request flow is:

![System flow](./mermaid-diagram.png)

                   +----------------------+
                   |      HTTP Client     |
                   | Postman / Frontend   |
                   +----------+-----------+
                              |
                              v
                   +----------------------+
                   |  delivery/http       |
                   |  Handler / Routes    |
                   +----------+-----------+
                              |
                              v
                   +----------------------+
                   |      usecase         |
                   |  Analyze Reviews     |
                   |  Analyze Steam Game  |
                   +-----+-----------+----+
                         |           |
                         |           |
                         v           v
             +----------------+   +----------------+
             | client/ai      |   | client/steam   |
             | Ollama Client  |   | Steam API      |
             +--------+-------+   +--------+-------+
                      |                    |
                      v                    v
             +----------------+   +----------------+
             | Local LLM       |   | Steam Reviews  |
             | Ollama / Model  |   | External API   |
             +----------------+   +----------------+

The service accepts a list of review texts and returns structured gameplay insights including:

- praised gameplay features
- common player complaints
- sentiment distribution
- a short AI-generated summary

This project demonstrates how LLMs can be integrated into backend systems to transform unstructured player feedback into actionable product insights.

## What It Does

This project is useful for turning raw player feedback into structured product insights from sources such as:

- Steam reviews
- App Store / Play Store reviews
- Discord or community feedback
- internal QA or survey comments

## Features

- REST API for manual review analysis
- REST API for Steam review analysis by `appId`
- Health check endpoint
- Ollama-based review analysis
- Environment-based configuration
- Modular internal structure with delivery, use case, and client layers

## Project Structure

```text
cmd/
  main.go

config/
  config.go

internal/
  prompt/
    review_prompt.go
  review/
    client/
      ai/
        client.go
        ollama.go
      steam/
        client.go
        steam.go
    delivery/
      http/
        handler.go
        response.go
        routes.go
    model/
      insight.go
      review.go
      steam.go
    usecase/
      analyze.go
```

## Requirements

- Go
- [Ollama](https://ollama.com)
- An Ollama model pulled locally, for example `llama3.2:3b`

## Configuration

The app reads environment variables from the system and also loads `.env` automatically on startup.

### Supported Variables

| Variable | Default | Description |
| --- | --- | --- |
| `SERVER_PORT` | `8080` | HTTP server port |
| `OLLAMA_BASE_URL` | `http://localhost:11434` | Ollama server URL |
| `OLLAMA_MODEL` | `llama3.2:3b` | Model used for `/analyze` |
| `OLLAMA_TIMEOUT_SEC` | `300` | Timeout for Ollama requests in seconds |
| `SUPABASE_DB_URL` | empty | Supabase Postgres connection string |
| `SUPABASE_DB_MAX_CONNS` | `5` | Maximum open database connections |
| `SUPABASE_DB_MIN_CONNS` | `0` | Minimum idle database connections |
| `SUPABASE_DB_HEALTH_TIMEOUT_SEC` | `5` | Timeout for database health checks in seconds |

Example `.env`:

```env
SERVER_PORT=8080
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2:3b
OLLAMA_TIMEOUT_SEC=300
SUPABASE_DB_URL=postgresql://postgres.your-project-ref:your-password@aws-0-your-region.pooler.supabase.com:5432/postgres
SUPABASE_DB_MAX_CONNS=5
SUPABASE_DB_MIN_CONNS=0
SUPABASE_DB_HEALTH_TIMEOUT_SEC=5
```

### Supabase Connection

For this long-running Go server, prefer the Supavisor session mode connection string from the Supabase dashboard (`Connect` -> `Session pooler`). Set it in `SUPABASE_DB_URL`.

If `SUPABASE_DB_URL` is empty, the API still starts and `/health` reports `"database": "disabled"`.

## Run Locally

### 1. Start Ollama

```bash
ollama serve
```

### 2. Pull a model

```bash
ollama pull llama3.2:3b
```

You can use another model, but then update `OLLAMA_MODEL` accordingly.

### 3. Run the API server

```bash
go run ./cmd
```

Server address:

```text
http://localhost:8080
```

## Run Unit Tests

This project includes unit tests for the HTTP delivery layer and the review use case.

Run all unit tests:

```bash
go test ./...
```

Run tests with verbose output:

```bash
go test -v ./...
```

Run tests for a specific package:

```bash
go test ./internal/review/usecase/...
go test ./internal/review/delivery/http/...
```

## API

### `GET /health`

Returns service status and database connectivity.

Response:

```json
{
  "status": "ok",
  "database": "connected"
}
```

When `SUPABASE_DB_URL` is not configured:

```json
{
  "status": "ok",
  "database": "disabled"
}
```

When the database is configured but unreachable, the endpoint returns `503 Service Unavailable` with:

```json
{
  "status": "degraded",
  "database": "unreachable"
}
```

### `POST /analyze`

Analyzes a list of player reviews.

Request body:

```json
{
  "reviews": [
    "Amazing combat and beautiful world.",
    "The game crashes too often after the latest patch.",
    "Great story, but performance is still bad."
  ]
}
```

Example response:

```json
{
  "review_count": 3,
  "praised_features": [
    "combat",
    "world design",
    "story"
  ],
  "common_issues": [
    "crashes",
    "performance issues"
  ],
  "topics": [
    "combat",
    "world design",
    "story",
    "performance"
  ],
  "sentiment": {
    "positive": 1,
    "neutral": 0,
    "negative": 2
  },
  "summary": "Players praise the combat, story, and world design, but recurring crashes and performance issues are hurting the experience."
}
```

### `POST /steam/analyze`

Fetches reviews from Steam by app ID, then analyzes them with the same Ollama flow.

Request body:

```json
{
  "appId": "730",
  "limit": 30,
  "language": "english"
}
```

Notes:

- `appId` is required
- `limit` defaults to `30` when omitted or invalid
- `language` defaults to `english` when omitted

Example response:

```json
{
  "review_count": 30,
  "praised_features": [
    "gunplay",
    "competitive gameplay"
  ],
  "common_issues": [
    "cheaters",
    "performance issues"
  ],
  "topics": [
    "gunplay",
    "matchmaking",
    "performance"
  ],
  "sentiment": {
    "positive": 18,
    "neutral": 4,
    "negative": 8
  },
  "summary": "Players still value the core gameplay, but complaints around cheating and performance continue to affect the overall experience."
}
```

## Error Cases

Typical API errors:

- `400 Bad Request` when request JSON is invalid
- `400 Bad Request` when `reviews` is missing, empty, or only contains blank strings
- `400 Bad Request` when `appId` is missing in `/steam/analyze`
- `405 Method Not Allowed` when using the wrong HTTP method
- `400 Bad Request` when Steam or Ollama analysis fails in the current handler flow

## Notes

- The current public API surface is `GET /health`, `POST /analyze`, and `POST /steam/analyze`.
- `.env.example` should stay in sync with [`config/config.go`](/Users/truongcongminh96/MinMin/Master%20of%20Science%20in%20Machine%20Learning%20and%20Artificial%20Intelligence/Learning/Projects/ai-game-review-analyzer/config/config.go).

## Example cURL

```bash
curl -X POST http://localhost:8080/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "reviews": [
      "Amazing combat and beautiful world.",
      "The game crashes too often after the latest patch.",
      "Great story, but performance is still bad."
    ]
  }'
```

Steam example:

```bash
curl -X POST http://localhost:8080/steam/analyze \
  -H "Content-Type: application/json" \
  -d '{
    "appId": "730",
    "limit": 30,
    "language": "english"
  }'
```

## License

MIT

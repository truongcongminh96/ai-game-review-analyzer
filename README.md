# AI Game Review Analyzer

A backend service written in Go that analyzes player reviews using a local LLM via Ollama.

The current request flow is:

`API -> analyze service -> Ollama client`

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

- REST API for submitting reviews
- Health check endpoint
- Ollama-based review analysis
- Environment-based configuration
- Modular internal structure for API, config, services, and AI client

## Project Structure

```text
cmd/
  main.go

internal/
  ai/
    ollama.go
  api/
    handler.go
  analyzer/
    review_analyzer.go
  config/
    config.go
  models/
    insight.go
    review.go
    sentiment.go
  report/
    report_generator.go
  service/
    analyze/
      analyze_service.go
    sentiment/
      sentiment_service.go
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
| `OPENAI_API_KEY` | empty | Optional, only used by `internal/report` |

Example `.env`:

```env
SERVER_PORT=8080
OLLAMA_BASE_URL=http://localhost:11434
OLLAMA_MODEL=llama3.2:3b
OPENAI_API_KEY=your_openai_api_key_here
```

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

## API

### `GET /health`

Returns service status.

Response:

```json
{
  "status": "ok"
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
  "praised_features": [
    "combat",
    "world design",
    "story"
  ],
  "common_issues": [
    "crashes",
    "performance issues"
  ],
  "sentiment": {
    "positive": 1,
    "neutral": 0,
    "negative": 2
  },
  "summary": "Players praise the combat, story, and world design, but recurring crashes and performance issues are hurting the experience."
}
```

## Error Cases

Typical API errors:

- `400 Bad Request` when request JSON is invalid
- `400 Bad Request` when `reviews` is missing, empty, or only contains blank strings
- `405 Method Not Allowed` when using the wrong HTTP method
- `500 Internal Server Error` when Ollama fails or returns unusable output

## Notes

- `internal/report/report_generator.go` contains an OpenAI-based report generator, but it is not currently wired into the HTTP API.
- `internal/analyzer` and `internal/service/sentiment` exist as separate logic modules and can be reused in future flows.
- `.env.example` should stay in sync with `internal/config/config.go`.

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

## License

MIT

# API v2

`v2` is the async, database-backed API surface for dashboard-style workflows.

Use `v1` when you want a simple synchronous response. Use `v2` when the client needs:

- a queued analysis run
- progress polling
- evidence drill-down
- run history
- comparison between runs

## Requirements

- Database migrations must be up to date
- Ollama must be running
- The configured model should be available locally

Run migrations:

```bash
go run ./cmd/migrate up
```

Start the server:

```bash
go run ./cmd
```

## Routes

### `POST /v2/steam/analyze`

Creates an analysis run and returns a `run_id` immediately.

Request body:

```json
{
  "appId": "1245620",
  "limit": 20,
  "language": "english"
}
```

Response `202 Accepted`:

```json
{
  "run_id": "3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a",
  "status": "pending",
  "current_stage": "queued",
  "progress_percent": 0,
  "request": {
    "app_id": "1245620",
    "limit": 20,
    "language": "english"
  },
  "links": {
    "self": "/v2/analysis-runs/3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a",
    "history": "/v2/games/1245620/history"
  }
}
```

### `GET /v2/analysis-runs/{runID}`

Returns the current run status or the final report.

Example response while still running:

```json
{
  "run_id": "3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a",
  "status": "pending",
  "current_stage": "analyzing",
  "progress_percent": 65,
  "requested_at": "2026-04-01T03:10:11Z",
  "started_at": "2026-04-01T03:10:11Z",
  "game": {
    "app_id": "1245620",
    "title": "ELDEN RING",
    "cover_url": "https://cdn.cloudflare.steamstatic.com/...",
    "genre": "Action RPG",
    "release_year": 2022
  },
  "overview": {
    "review_count": 0,
    "sentiment": {
      "positive": 0,
      "neutral": 0,
      "negative": 0
    },
    "summary": ""
  },
  "praises": [],
  "issues": [],
  "topics": []
}
```

Example response after completion:

```json
{
  "run_id": "3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a",
  "status": "success",
  "current_stage": "completed",
  "progress_percent": 100,
  "requested_at": "2026-04-01T03:10:11Z",
  "started_at": "2026-04-01T03:10:11Z",
  "completed_at": "2026-04-01T03:10:19Z",
  "game": {
    "app_id": "1245620",
    "title": "ELDEN RING",
    "cover_url": "https://cdn.cloudflare.steamstatic.com/...",
    "genre": "Action RPG",
    "release_year": 2022
  },
  "overview": {
    "review_count": 20,
    "sentiment": {
      "positive": 14,
      "neutral": 1,
      "negative": 5
    },
    "summary": "Players strongly praise exploration and combat depth, while criticism centers on performance and occasional difficulty spikes."
  },
  "praises": [
    {
      "id": "item-1",
      "kind": "praise",
      "label": "exploration",
      "summary": "Players consistently praise the sense of discovery and world design.",
      "confidence": 0.93,
      "evidence_count": 3,
      "sample_evidence": [
        {
          "review_id": "rev-1",
          "quote": "every area feels worth exploring",
          "voted_up": true,
          "language": "english",
          "helpful_votes": 12,
          "funny_votes": 0,
          "playtime_hours": 48.5
        }
      ]
    }
  ],
  "issues": [
    {
      "id": "item-2",
      "kind": "issue",
      "label": "performance",
      "summary": "Some players report stuttering and frame drops in open areas.",
      "severity": 4,
      "confidence": 0.88,
      "evidence_count": 2,
      "sample_evidence": [
        {
          "review_id": "rev-2",
          "quote": "stutters badly in some open world zones",
          "voted_up": false,
          "language": "english",
          "helpful_votes": 9,
          "funny_votes": 1,
          "playtime_hours": 31.2
        }
      ]
    }
  ],
  "topics": [
    {
      "id": "item-3",
      "kind": "topic",
      "label": "build variety",
      "summary": "Many reviews discuss experimenting with different builds and weapons.",
      "confidence": 0.81,
      "evidence_count": 2,
      "sample_evidence": [
        {
          "review_id": "rev-3",
          "quote": "tons of viable builds to try",
          "voted_up": true,
          "language": "english",
          "helpful_votes": 7,
          "funny_votes": 0,
          "playtime_hours": 60.0
        }
      ]
    }
  ]
}
```

### `GET /v2/analysis-runs/{runID}/evidence`

Query params:

- `kind`: `issue`, `praise`, or `topic`
- `label`: exact label to inspect
- `limit`: optional, defaults to `20`

Example:

```text
GET /v2/analysis-runs/3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a/evidence?kind=issue&label=performance
```

Response:

```json
{
  "items": [
    {
      "review_id": "rev-2",
      "quote": "stutters badly in some open world zones",
      "review_text": "Amazing game overall, but it stutters badly in some open world zones on my PC...",
      "voted_up": false,
      "language": "english",
      "helpful_votes": 9,
      "funny_votes": 1,
      "playtime_hours": 31.2,
      "reviewed_at": "2026-03-28T08:10:00Z"
    }
  ]
}
```

### `GET /v2/games/{appID}/history`

Returns recent runs for a Steam app.

Response:

```json
{
  "game": {
    "app_id": "1245620",
    "title": "ELDEN RING",
    "cover_url": "https://cdn.cloudflare.steamstatic.com/...",
    "genre": "Action RPG",
    "release_year": 2022
  },
  "items": [
    {
      "run_id": "3f6be8fd-7e87-4b90-9f0f-bd3cb17f6c4a",
      "requested_at": "2026-04-01T03:10:11Z",
      "review_count": 20,
      "sentiment": {
        "positive": 14,
        "neutral": 1,
        "negative": 5
      },
      "summary": "Players strongly praise exploration and combat depth, while criticism centers on performance and occasional difficulty spikes."
    }
  ]
}
```

### `GET /v2/compare`

Query params:

- `runA`
- `runB`

Example:

```text
GET /v2/compare?runA=run-old&runB=run-new
```

Response:

```json
{
  "run_a": {
    "run_id": "run-old",
    "label": "Previous"
  },
  "run_b": {
    "run_id": "run-new",
    "label": "Current"
  },
  "summary": "Positive +2, neutral -1, negative -1. Issue change: performance is new.",
  "sentiment_delta": {
    "positive": 2,
    "neutral": -1,
    "negative": -1
  },
  "issues": [
    {
      "label": "performance",
      "change": "new"
    }
  ]
}
```

## Recommended Client Flow

1. `POST /v2/steam/analyze`
2. Poll `GET /v2/analysis-runs/{runID}` until status becomes `success` or `failed`
3. Render `overview`, `praises`, `issues`, and `topics`
4. Open `GET /v2/analysis-runs/{runID}/evidence` when the user drills into an item
5. Use `GET /v2/games/{appID}/history` for timelines
6. Use `GET /v2/compare` for current-vs-previous analysis

## Notes

- `v2` depends on the database-backed persistence layer
- `v1` and `v2` are both supported right now
- Larger Ollama models can make runs stay in `analyzing` for noticeably longer

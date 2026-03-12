# 🎮 AI Game Review Analyzer

AI-powered backend system that analyzes player feedback from games and extracts insights about gameplay systems, sentiment, and common complaints.

This project demonstrates how AI + backend services can help game studios understand player feedback at scale.

---

Analyze player feedback and extract gameplay insights using LLMs.

Features
- Sentiment analysis
- Gameplay topic extraction
- Issue detection
- AI-generated summary

Tech Stack
- Go backend
- Ollama (Qwen LLM)
- REST API

# Architecture
cmd/
main.go

internal/
api/
handler.go

analyzer/
review_analyzer.go

sentiment/
sentiment_service.go

report/
report_generator.go

models/
review.go

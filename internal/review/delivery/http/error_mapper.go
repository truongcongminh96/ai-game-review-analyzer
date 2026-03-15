package http

import (
	"errors"
	"net/http"
	"strings"
)

func mapErrorToStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}

	msg := err.Error()

	switch {
	case strings.Contains(msg, "invalid request"),
		strings.Contains(msg, "required"),
		strings.Contains(msg, "cannot be empty"):
		return http.StatusBadRequest
	case strings.Contains(msg, "steam returned status"),
		strings.Contains(msg, "failed to call steam"),
		strings.Contains(msg, "ollama returned status"),
		strings.Contains(msg, "failed to call ollama"):
		return http.StatusBadGateway
	default:
		var target interface{ Temporary() bool }
		if errors.As(err, &target) {
			return http.StatusBadGateway
		}
		return http.StatusInternalServerError
	}
}

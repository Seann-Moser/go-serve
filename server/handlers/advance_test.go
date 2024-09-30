package handlers

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

// Mock Ping type function that returns true or false based on health

// Test NewAdvancedHealthCheck function with all pings successful
func TestNewAdvancedHealthCheck_AllPingsSuccess(t *testing.T) {
	// Arrange
	pings := map[string]Ping{
		"db":  func(ctx context.Context) bool { return true },
		"api": func(ctx context.Context) bool { return true },
	}

	timeout := time.Second
	endpoint := NewAdvancedHealthCheck(timeout, pings)

	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	// Act
	endpoint.HandlerFunc(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String())
}

// Test NewAdvancedHealthCheck function with one ping failure
func TestNewAdvancedHealthCheck_OnePingFailure(t *testing.T) {
	// Arrange
	pings := map[string]Ping{
		"db":  func(ctx context.Context) bool { return false }, // Fails
		"api": func(ctx context.Context) bool { return true },
	}

	timeout := time.Second
	endpoint := NewAdvancedHealthCheck(timeout, pings)

	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	// Act
	endpoint.HandlerFunc(rr, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to ping:db")
}

// Test NewAdvancedHealthCheck function with multiple ping failures
func TestNewAdvancedHealthCheck_MultiplePingFailures(t *testing.T) {
	// Arrange
	pings := map[string]Ping{
		"db":  func(ctx context.Context) bool { return false }, // Fails
		"api": func(ctx context.Context) bool { return false }, // Fails
	}

	timeout := time.Second
	endpoint := NewAdvancedHealthCheck(timeout, pings)

	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	// Act
	endpoint.HandlerFunc(rr, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to ping:db")
	assert.Contains(t, rr.Body.String(), "failed to ping:api")
}

// Test NewAdvancedHealthCheck with context timeout before ping execution
func TestNewAdvancedHealthCheck_ContextTimeout(t *testing.T) {
	// Arrange
	pings := map[string]Ping{
		"db": func(ctx context.Context) bool {
			select {
			case <-ctx.Done():
				return false // Simulate timeout
			case <-time.After(2 * time.Second): // Simulate long ping
				return true
			}
		},
	}

	timeout := 1 * time.Millisecond // Short timeout
	endpoint := NewAdvancedHealthCheck(timeout, pings)

	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	// Act
	endpoint.HandlerFunc(rr, req)

	// Assert
	assert.Equal(t, http.StatusServiceUnavailable, rr.Code)
	assert.Contains(t, rr.Body.String(), "failed to ping:db")
}

// Test NewAdvancedHealthCheck with no pings (edge case)
func TestNewAdvancedHealthCheck_NoPings(t *testing.T) {
	// Arrange
	pings := map[string]Ping{}
	timeout := time.Second
	endpoint := NewAdvancedHealthCheck(timeout, pings)

	req, _ := http.NewRequest(http.MethodGet, "/healthcheck", nil)
	rr := httptest.NewRecorder()

	// Act
	endpoint.HandlerFunc(rr, req)

	// Assert
	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Empty(t, rr.Body.String())
}

package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// Total requests counter
	// Tracks total number of requests by upstream and status (success/error)
	requestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "proxy_requests_total",
			Help: "Total HTTP requests processed by the proxy",
		},
		[]string{"upstream", "status"},
	)

	// Request latency histogram
	// Tracks distribution of request latencies in milliseconds
	requestLatency = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "proxy_request_latency_ms",
			Help: "Request latency in milliseconds",
			Buckets: []float64{10, 50, 100, 500, 1000, 5000},
		},
		[]string{"upstream"},
	)

	// Circuit breaker state gauge
	// Tracks current state of circuit breaker per upstream
	// 0 = closed (healthy), 1 = open (down), 2 = half-open (testing)
	circuitBreakerState = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "proxy_circuit_breaker_state",
			Help: "Circuit breaker state per upstream (0=closed, 1=open, 2=half-open)",
		},
		[]string{"upstream"},
	)
)

// init() runs automatically on startup and registers all metrics
func init() {
	prometheus.MustRegister(requestsTotal)
	prometheus.MustRegister(requestLatency)
	prometheus.MustRegister(circuitBreakerState)
}

// RecordRequest records a successful request to metrics
func RecordRequest(upstream string, latencyMs int64) {
	requestsTotal.WithLabelValues(upstream, "success").Inc()
	requestLatency.WithLabelValues(upstream).Observe(float64(latencyMs))
}

// RecordError records a failed request to metrics
func RecordError(upstream string, latencyMs int64) {
	requestsTotal.WithLabelValues(upstream, "error").Inc()
	requestLatency.WithLabelValues(upstream).Observe(float64(latencyMs))
}

// UpdateCircuitBreakerState updates the circuit breaker state gauge
func UpdateCircuitBreakerState(upstream string, state string) {
	var stateValue float64

	switch state {
	case "closed":
		stateValue = 0
	case "open":
		stateValue = 1
	case "half-open":
		stateValue = 2
	default:
		stateValue = 0
	}

	circuitBreakerState.WithLabelValues(upstream).Set(stateValue)
}
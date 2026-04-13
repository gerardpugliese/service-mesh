package main

import (
	"fmt"
	"net/http"
	"io"
	"io/ioutil"
	"sync"
	"time"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"context"
	"errors"
	"crypto/tls"
	"crypto/x509"
)

/*  

	GOAL: Build a proxy that listens on :8080, accepts requests, and forwards them to an upstream service on :3000

	Proxy Requirements:
		1. Listen on port 8080
		2. Wehn it gets a request from a client, forward that request to the upstream service on :3000
		3. Get the response from the upstream
		4. Send that response back to the client
*/

type LoadBalancer struct {
	upstreams []string
	breakers map[string]*CircuitBreaker
	counter int
	mu sync.Mutex
	requestTimeout time.Duration // How long to wait for the upstream
}

// Load Balancer method for selecting an upstream server to send a request to
func (lb *LoadBalancer) SelectUpstream() string {
	// Lock LoadBalancer
	lb.mu.Lock()
	defer lb.mu.Unlock()
	selected := lb.upstreams[lb.counter % len(lb.upstreams)]
	// Increment counter
	lb.counter++

	return selected
}

func (lb *LoadBalancer) handleRequest(w http.ResponseWriter, r *http.Request) {
	// Deserialize request
	fmt.Printf("Proxy got: %s %s\n", r.Method, r.URL.Path)

	// Start timer for latency measurement
	start := time.Now()

	// Selecting upstream to forward to
	selectedUpstream := lb.SelectUpstream()
	fmt.Printf("Forwarding to: %s\n", selectedUpstream)

	// Create a Context with a 5-second timeout
	ctx, cancel := context.WithTimeout(context.Background(), lb.requestTimeout)
	defer cancel()
	
	
	// Get CircuitBreaker for this upstream
	cb := lb.breakers[selectedUpstream]

	state := cb.GetState()
	failures := cb.GetFailureCount()
	fmt.Printf("Circuit state: %s (failures: %d)\n", state, failures)

	// If upstream is open, return error
	if cb.IsOpen() {
		fmt.Println("Upstream is unavailable.")

		// Record error metric
		latency := time.Since(start).Milliseconds()
		RecordError(selectedUpstream, latency)

		// Update circuit breaker state metric
		UpdateCircuitBreakerState(selectedUpstream, cb.GetState())

		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Upstream service unavailable: " + selectedUpstream))
		return
	}

	// Construct full URL for the request
	upstreamURL := selectedUpstream + r.RequestURI

	// Create request for Upstream
	req, err := http.NewRequestWithContext(ctx, r.Method, upstreamURL, r.Body)
	if err != nil {
		fmt.Println("Error creating request: ", err, "\n")
	
		// Record error metric
		latency := time.Since(start).Milliseconds()
		RecordError(selectedUpstream, latency)

		return
	}

	/* TLS CONFIGURATION */

	// Load client certificate and key
	clientCert, err := tls.LoadX509KeyPair("../certs/client-cert.pem", "../certs/client-key.pem")
	if err != nil {
		panic(err)
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile("../certs/ca-cert.pem")
	if err != nil {
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Create TLS config
	tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{clientCert},
			RootCAs: caCertPool,
		}

	// Copy headers from client request to upstream request
	req.Header = r.Header

	// Create an HTTP client
	client := &http.Client {
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		// Record error in CircuitBreaker
		cb.RecordFailure()

		//Check if it's a timeout
		if errors.Is(err, context.DeadlineExceeded) {
			fmt.Printf("Timeout calling upstream: %v\n", err)
			latency := time.Since(start).Milliseconds()
			RecordTimeout(selectedUpstream)
			RecordError(selectedUpstream, latency)
			UpdateCircuitBreakerState(selectedUpstream, cb.GetState())

			w.WriteHeader(http.StatusGatewayTimeout)
			w.Write([]byte("Upstream request timeout"))
			return
		}

		fmt.Printf("Error calling upstream: %v\n", err)

		// Record error metric
		latency := time.Since(start).Milliseconds()
		RecordError(selectedUpstream, latency)

		// Update circuit breaker state metric
		UpdateCircuitBreakerState(selectedUpstream, cb.GetState())

		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Upstream service unavailable"))
		return
	} 

	// Record success in CircuitBreaker
	cb.RecordSuccess()

	// Record success metric
	latency := time.Since(start).Milliseconds()
	RecordRequest(selectedUpstream, latency)

	// Update circuit breaker state metric
	UpdateCircuitBreakerState(selectedUpstream, cb.GetState())
	
	// Copy the response headers from upstream back to the client
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Copy the HTTP status code from upstream
	w.WriteHeader(resp.StatusCode)

	// Copy the response body from upstream to the client
	io.Copy(w, resp.Body)
	resp.Body.Close()

	fmt.Printf("Response sent to client\n")
}


func main() {
	// Create Load Balancer
	lb := &LoadBalancer{
		upstreams: []string{
			"https://localhost:3003",
			"https://localhost:3001",
			"https://localhost:3002",
		},
		breakers: map[string]*CircuitBreaker{
			"https://localhost:3003": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
			"https://localhost:3001": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
			"https://localhost:3002": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
		},
		counter: 0,
		requestTimeout: 	5 * time.Second,
	}

	// Add metrics endpoint
	http.Handle("/metrics", promhttp.Handler())

	// Assign HTTP function to '/' path
	http.HandleFunc("/", lb.handleRequest)

	// Set up listener on port 8080
	fmt.Printf("Upstream service listening on :8080\n")
	http.ListenAndServe(":8080", nil)
}
package main

import (
	"fmt"
	"net/http"
	"io"
	"sync"
	"time"
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

	// Selecting upstream to forward to
	selectedUpstream := lb.SelectUpstream()
	fmt.Printf("Forwarding to: %s\n", selectedUpstream)
	
	// Get CircuitBreaker for this upstream
	cb := lb.breakers[selectedUpstream]

	state = cb.GetState()
	failures = cb.GetFailureCount()
	fmt.Printf("Circuit state: %s (failures: %d)\n", state, failures)

	// If upstream is open, return error
	if cb.IsOpen() {
		fmt.Println("Upstream is unavailable.")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte("Upstream service unavailable: " + selectedUpstream))
		return
	}

	// Construct full URL for the request
	upstreamURL := selectedUpstream + r.RequestURI

	// Create request for Upstream
	req, err := http.NewRequest(r.Method, upstreamURL, r.Body)
	if err != nil {
		fmt.Println("Error creating request: ", err, "\n")
		return
	}

	// Copy headers from client request to upstream request
	req.Header = r.Header

	// Create an HTTP client
	client := &http.Client {}

	// Execute request
	resp, err := client.Do(req)
	if err != nil {
		// Record error in CircuitBreaker
		cb.RecordFailure()
		fmt.Printf("Error calling upstream: %v\n", err)
		w.WriteHeader(http.StatusBadGateway)
		w.Write([]byte("Upstream service unavailable"))
		return
	} 

	// Record success in CircuitBreaker
	cb.RecordSuccess()
	
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
			"http://localhost:3000",
			"http://localhost:3001",
			"http://localhost:3002",
		},
		breakers: map[string]*CircuitBreaker{
			"http://localhost:3000": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
			"http://localhost:3001": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
			"http://localhost:3002": &CircuitBreaker{
					failureThreshold:  	3,
					timeout: 			5 * time.Second,
					state: 				"closed",
				},
		},
		counter: 0,
	}

	// Assign HTTP function to '/' path
	http.HandleFunc("/", lb.handleRequest)

	// Set up listener on port 8080
	fmt.Printf("Upstream service listening on :8080\n")
	http.ListenAndServe(":8080", nil)
}
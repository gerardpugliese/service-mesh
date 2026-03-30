package main

import (
	"fmt"
	"net/http"
	"time"
)

/*  

	GOAL: Build a proxy that listens on :8080, accepts requests, and forwards them to an upstream service on :3000

	Upstream Requirements:
		1. Listen on port 3000
		2. When it receives a request, print what it got: `Upstream got: GET /api/users`
		3. Send back a response: `"Response from upstream service!`

*/

func handleRequest (w http.ResponseWriter, r *http.Request) {
	// Deserialize request
	fmt.Printf("Upstream #2 got: %s %s\n", r.Method, r.RequestURI)
	time.Sleep(10 * time.Second)
	w.Write([]byte("Response from upstream 2\n"))
}


func main() {
	// Assign HTTP function to '/' path
	http.HandleFunc("/", handleRequest)

	// Set up listener on port 3001
	fmt.Printf("Upstream service listening on :3001\n")
	http.ListenAndServe(":3001", nil)
}
package main

import (
	"fmt"
	"net/http" 
	"crypto/tls"
	"crypto/x509"
	"io/ioutil"
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
	w.Write([]byte("Response from upstream 2\n"))
}


func main() {
	// Assign HTTP function to '/' path
	http.HandleFunc("/", handleRequest)

	// Set up listener on port 3001
	fmt.Printf("Upstream service listening on :3001\n")
	
	// Load certificate and key
	cert, err := tls.LoadX509KeyPair("../certs/server-cert.pem", "../certs/server-key.pem")
	if err != nil {
		panic(err)
	}

	// Load CA certificate
	caCert, err := ioutil.ReadFile("../certs/ca-cert.pem")
	if err != nil {
		panic(err)
	}

	caCertPool := x509.NewCertPool()
	if !caCertPool.AppendCertsFromPEM(caCert) {
        panic("failed to parse CA certificate")
    }

	// Create TLS config
	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		ClientAuth: tls.RequireAndVerifyClientCert,  // Require client certs
   	 	ClientCAs: caCertPool,  // Trust certs signed by this CA
	}

	// Create TLS listener
	listener, err := tls.Listen("tcp", ":3001", tlsConfig)
	if err != nil {
		panic(err)
	}

	// Serve HTTPS
	server := &http.Server{Handler: http.DefaultServeMux}
	server.Serve(listener)
}
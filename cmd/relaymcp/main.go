package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	var (
		port   = flag.Int("port", 8080, "Port to listen on")
		target = flag.String("target", "127.0.0.1:3845", "Target host:port to relay to")
		help   = flag.Bool("help", false, "Show help")
	)
	flag.Parse()

	if *help {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "\nOptions:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -port 8080 -target 127.0.0.1:3845\n", os.Args[0])
		os.Exit(0)
	}

	// Parse target URL
	targetURL, err := url.Parse("http://" + *target)
	if err != nil {
		log.Fatalf("Invalid target URL: %v", err)
	}

	// Create relay handler
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("Relaying %s %s to %s", r.Method, r.URL.Path, targetURL.Host)
		
		// Check if this is an SSE request
		if isSSERequest(r) {
			handleSSE(w, r, targetURL)
		} else {
			handleHTTP(w, r, targetURL)
		}
	})

	addr := "0.0.0.0:" + strconv.Itoa(*port)
	log.Printf("Starting relay server on port %d", *port)
	log.Printf("Relaying all requests to %s", targetURL.Host)

	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}

// isSSERequest checks if the request is for Server-Sent Events
func isSSERequest(r *http.Request) bool {
	return r.Method == "GET" && strings.Contains(r.Header.Get("Accept"), "text/event-stream")
}

// handleHTTP handles regular HTTP requests
func handleHTTP(w http.ResponseWriter, r *http.Request, targetURL *url.URL) {
	// Create target request
	targetReq, err := http.NewRequestWithContext(r.Context(), r.Method, "", r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set target URL
	targetReq.URL.Scheme = targetURL.Scheme
	targetReq.URL.Host = targetURL.Host
	targetReq.URL.Path = r.URL.Path
	targetReq.URL.RawQuery = r.URL.RawQuery
	targetReq.Host = targetURL.Host

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			targetReq.Header.Add(key, value)
		}
	}

	// Make request
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(targetReq)
	if err != nil {
		log.Printf("HTTP proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Copy response headers
	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Copy response body
	_, err = io.Copy(w, resp.Body)
	if err != nil {
		log.Printf("HTTP copy error: %v", err)
	}
}

// handleSSE handles Server-Sent Events requests
func handleSSE(w http.ResponseWriter, r *http.Request, targetURL *url.URL) {
	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	// Create target request
	targetReq, err := http.NewRequestWithContext(ctx, r.Method, "", r.Body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	// Set target URL
	targetReq.URL.Scheme = targetURL.Scheme
	targetReq.URL.Host = targetURL.Host
	targetReq.URL.Path = r.URL.Path
	targetReq.URL.RawQuery = r.URL.RawQuery
	targetReq.Host = targetURL.Host

	// Copy headers
	for key, values := range r.Header {
		for _, value := range values {
			targetReq.Header.Add(key, value)
		}
	}

	// Make request with no timeout for SSE
	client := &http.Client{Timeout: 0}
	resp, err := client.Do(targetReq)
	if err != nil {
		log.Printf("SSE proxy error: %v", err)
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	// Copy other response headers
	for key, values := range resp.Header {
		if key != "Content-Type" && key != "Cache-Control" && key != "Connection" {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}
	}

	// Write status code
	w.WriteHeader(resp.StatusCode)

	// Flush headers immediately
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}

	// Stream response
	reader := bufio.NewReader(resp.Body)
	for {
		select {
		case <-ctx.Done():
			log.Printf("SSE client disconnected")
			return
		default:
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err != io.EOF {
					log.Printf("SSE stream error: %v", err)
				}
				return
			}

			// Write line to client
			_, err = w.Write(line)
			if err != nil {
				log.Printf("SSE write error: %v", err)
				return
			}

			// Flush after each line
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
		}
	}
}
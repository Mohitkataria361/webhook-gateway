package main

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
)

// MockReceiver simulates three webhook receiver behaviors:
//   - /ok      → Always returns 200 OK
//   - /fail    → Always returns 500 Internal Server Error
//   - /timeout → Delays 30s (simulates a timeout)
//   - /flaky   → Returns 500 for the first N calls, then 200
//
// Usage:
//
//	go run tests/mock_receiver/main.go
//	# Server starts on :9090
func main() {
	port := "9090"
	if p := os.Getenv("MOCK_PORT"); p != "" {
		port = p
	}

	// Request counter for flaky endpoint.
	callCount := 0

	mux := http.NewServeMux()

	// --- /ok: Always succeeds ---
	mux.HandleFunc("/ok", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		sig := r.Header.Get("X-Signature-256")
		log.Printf("[mock/ok] Received: sig=%s payload=%s", sig, string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, `{"status":"received"}`)
	})

	// --- /fail: Always returns 500 ---
	mux.HandleFunc("/fail", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		log.Printf("[mock/fail] Received payload: %s", string(body))
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintln(w, `{"error":"internal server error"}`)
	})

	// --- /timeout: Sleeps to trigger client timeout ---
	mux.HandleFunc("/timeout", func(w http.ResponseWriter, r *http.Request) {
		log.Println("[mock/timeout] Sleeping 35s to trigger client timeout...")
		time.Sleep(35 * time.Second)
		w.WriteHeader(http.StatusGatewayTimeout)
	})

	// --- /flaky: Returns 500 for first N calls, then 200 ---
	mux.HandleFunc("/flaky", func(w http.ResponseWriter, r *http.Request) {
		failFor := 3 // fail first 3 calls
		if envVal := os.Getenv("FLAKY_FAIL_COUNT"); envVal != "" {
			if n, err := strconv.Atoi(envVal); err == nil {
				failFor = n
			}
		}

		callCount++
		log.Printf("[mock/flaky] Call #%d (failFor=%d)", callCount, failFor)

		if callCount <= failFor {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, `{"error":"temporary failure","call":%d}`, callCount)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok","call":%d}`, callCount)
	})

	// --- /reset: Resets flaky counter (for tests) ---
	mux.HandleFunc("/reset", func(w http.ResponseWriter, r *http.Request) {
		callCount = 0
		log.Println("[mock/reset] Call counter reset")
		fmt.Fprintln(w, `{"reset":true}`)
	})

	// --- Health check ---
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `{"status":"ok"}`)
	})

	addr := ":" + port
	log.Printf("[mock-receiver] Listening on %s", addr)
	log.Printf("[mock-receiver] Endpoints: /ok  /fail  /timeout  /flaky  /reset  /health")

	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

package main

import (
	"encoding/json"
	"io"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

type SpryngResponse struct {
	Code        int    `json:"code"`
	Description string `json:"description"`
}

type SpryngCallback struct {
	Status    string `json:"status"`
	Reference string `json:"reference"`
}

var (
	SPRYNG_MIN_DELAY_MS int
	SPRYNG_MAX_DELAY_MS int
	SPRYNG_CALLBACK_URL string
	spryngClient        *http.Client
)
func init() {
	// Load config
	SPRYNG_MIN_DELAY_MS, _ = strconv.Atoi(getenv("SPRYNG_MIN_DELAY_MS", "100"))
	SPRYNG_MAX_DELAY_MS, _ = strconv.Atoi(getenv("SPRYNG_MAX_DELAY_MS", "1000"))
	SPRYNG_CALLBACK_URL = getenv("SPRYNG_CALLBACK_URL", "http://localhost:6011/notifications/sms/spryng")
	maxConns, _ := strconv.Atoi(getenv("SPRYNG_MAX_CONNS", "256"))

	// Setup HTTP client
	
	spryngClient = &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxConnsPerHost: maxConns,
		},
	}

	log.Printf(
		"Spryng callback configured → URL: %s | delay: %d–%d ms | maxConns: %d\n",
		SPRYNG_CALLBACK_URL, SPRYNG_MIN_DELAY_MS, SPRYNG_MAX_DELAY_MS, maxConns,
	)
}


// SpryngEndpoint receives and logs incoming messages from Spryng-like services
func SpryngEndpoint(w http.ResponseWriter, r *http.Request) {
	log.Println("==== Incoming Spryng Request ====")
	log.Printf("Method: %s | URL: %s", r.Method, r.URL.String())

	// Log headers
	log.Println("Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	// Read and log body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusInternalServerError)
		log.Printf("Error reading body: %v", err)
		return
	}
	defer r.Body.Close()

	log.Println("Body:")
	log.Println(string(bodyBytes))
	log.Println("=================================")

	// Reconstruct body for further parsing
	r.Body = io.NopCloser(strings.NewReader(string(bodyBytes)))

	// Parse form if applicable
	if err := r.ParseForm(); err != nil {
		http.Error(w, "Failed to parse form", http.StatusInternalServerError)
		log.Printf("ParseForm error: %v", err)
		return
	}

	// Extract values (optional, uncomment if needed)
	// reference := r.FormValue("reference")
	// message := r.FormValue("message")
	// mobile := r.FormValue("mobile")
	// log.Printf("Parsed form values → reference=%s | message=%s | mobile=%s", reference, message, mobile)

	// Respond
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(SpryngResponse{
		Code:        0,
		Description: "SMS successfully queued",
	})
}

// SpryngSendCallback sends a delayed callback to simulate provider behavior
func SpryngSendCallback(reference, to string) {
	// Create a local random generator with a non-deterministic seed
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// Calculate a random delay between min and max
	delay := SPRYNG_MIN_DELAY_MS
	if SPRYNG_MAX_DELAY_MS > SPRYNG_MIN_DELAY_MS {
		delay += r.Intn(SPRYNG_MAX_DELAY_MS - SPRYNG_MIN_DELAY_MS)
	}

	time.Sleep(time.Duration(delay) * time.Millisecond)

	// Prepare callback data
	data := url.Values{
		"status":    {"0"},
		"reference": {reference},
		"mobile":    {to},
	}

	req, err := http.NewRequest("POST", SPRYNG_CALLBACK_URL, strings.NewReader(data.Encode()))
	if err != nil {
		log.Printf("Failed to create callback request: %v", err)
		return
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	if os.Getenv("USE_ECS_APPS") == "true" {
		req.Header.Set("x-notify-ecs-origin", "true")
	}

	res, err := spryngClient.Do(req)
	if err != nil {
		log.Printf("Spryng callback failed for %s: %v", reference, err)
		return
	}
	
	defer res.Body.Close()

	log.Printf("Spryng callback sent for %s → %s", reference, res.Status)
}


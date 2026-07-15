package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"time"
)

type SpryngRequest struct {
	Originator string   `json:"originator"`
	Recipients []string `json:"recipients"`
	Body       string   `json:"body"`
	Reference  string   `json:"reference"`
	Route      string   `json:"route"`
	Encoding   string   `json:"encoding"`
}

type SpryngResponse struct {
	Description string `json:"description"`
	Code        int    `json:"code"`
}

var SPRYNG_MIN_DELAY_MS int
var SPRYNG_MAX_DELAY_MS int
var SPRYNG_CALLBACK_URL string
var spryngClient *http.Client

func init() {
	SPRYNG_MIN_DELAY_MS, _ = strconv.Atoi(getenv("SPRYNG_MIN_DELAY_MS", "100"))
	SPRYNG_MAX_DELAY_MS, _ = strconv.Atoi(getenv("SPRYNG_MAX_DELAY_MS", "1000"))
	SPRYNG_CALLBACK_URL = getenv("SPRYNG_CALLBACK_URL", "http://localhost:6011/notifications/sms/spryng")
	var maxConns, _ = strconv.Atoi(getenv("SPRYNG_MAX_CONNS", "256"))

	log.Printf("Spryng callback: URL %s, with delay %d-%d ms\n", SPRYNG_CALLBACK_URL, SPRYNG_MIN_DELAY_MS, SPRYNG_MAX_DELAY_MS)

	spryngClient = &http.Client{
		Timeout: time.Second * 10,
		Transport: &http.Transport{
			MaxConnsPerHost: maxConns,
		},
	}
}

func SpryngEndpoint(w http.ResponseWriter, r *http.Request) {
	var reqData SpryngRequest
	if err := json.NewDecoder(r.Body).Decode(&reqData); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	log.Printf("Spryng received message %s:\n\n    %s\n\n", reqData.Reference, reqData.Body)

	json.NewEncoder(w).Encode(SpryngResponse{Code: 0, Description: "SMS successfully queued"})
	go SpryngSendCallback(reqData.Reference)
}

func SpryngSendCallback(reference string) {
	time.Sleep(time.Duration(SPRYNG_MIN_DELAY_MS+rand.Intn(SPRYNG_MAX_DELAY_MS-SPRYNG_MIN_DELAY_MS)) * time.Millisecond)

	params := url.Values{
		"STATUS":     {"10"},
		"REASONCODE": {"0"},
		"REFERENCE":  {reference},
	}

	req, err := http.NewRequest("GET", SPRYNG_CALLBACK_URL+"?"+params.Encode(), nil)
	if err != nil {
		log.Printf("Spryng callback request build failed: %s\n", err.Error())
		return
	}

	ecs_header := os.Getenv("USE_ECS_APPS")
	if ecs_header == "true" {
		req.Header.Set("x-notify-ecs-origin", "true")
	}

	res, err := spryngClient.Do(req)

	if err != nil {
		log.Printf("Spryng callback failed: %s\n", err.Error())
		return
	}

	res.Body.Close()

	log.Printf("Spryng callback sent for %s: %s", reference, res.Status)
}

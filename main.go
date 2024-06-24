package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/gocolly/colly/v2"
)

type ScrapeRequest struct {
	URL         string            `json:"url"`
	Headers     map[string]string `json:"headers"`
	Selector    string            `json:"selector"`
	ContentType string            `json:"contentType"`
}

type ValidateResponse struct {
	Success bool `json:"success"`
}

type ScrapeResponse struct {
	Data []string `json:"data"`
}

func main() {
	http.HandleFunc("/validateSelector", handleValidateSelector)
	http.HandleFunc("/scrape", handleScrape)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func setCORSHeaders(w http.ResponseWriter) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "POST, GET, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
}

func handleValidateSelector(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	c := colly.NewCollector(
		colly.AllowURLRevisit(),
	)

	for key, value := range req.Headers {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(key, value)
		})
	}

	success := false
	c.OnHTML(req.Selector, func(e *colly.HTMLElement) {
		success = true
	})

	err := c.Visit(req.URL)
	if err != nil {
		success = false
	}

	response := ValidateResponse{
		Success: success,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleScrape(w http.ResponseWriter, r *http.Request) {
	setCORSHeaders(w)
	if r.Method == http.MethodOptions {
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ScrapeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	c := colly.NewCollector(
		colly.AllowURLRevisit(),
	)

	for key, value := range req.Headers {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(key, value)
		})
	}

	var scrapedData []string
	c.OnHTML(req.Selector, func(e *colly.HTMLElement) {
		switch strings.ToLower(req.ContentType) {
		case "text":
			scrapedData = append(scrapedData, e.Text)
		case "title":
			scrapedData = append(scrapedData, e.ChildText("title"))
		case "href":
			href := e.Attr("href")
			scrapedData = append(scrapedData, href)
		default:
			scrapedData = append(scrapedData, e.Text)
		}
	})

	err := c.Visit(req.URL)
	if err != nil {
		http.Error(w, "Failed to visit URL", http.StatusInternalServerError)
		return
	}

	response := ScrapeResponse{
		Data: scrapedData,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

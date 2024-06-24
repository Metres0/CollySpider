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
	DataTypes   []string          `json:"dataTypes"`
	NextPageURL string            `json:"nextPageUrl"`
}

type ValidateResponse struct {
	Success   bool     `json:"success"`
	DataTypes []string `json:"dataTypes"`
	Error     string   `json:"error,omitempty"`
}

type ScrapeResponse struct {
	Data        []ScrapedData `json:"data"`
	NextPageURL string        `json:"nextPageUrl,omitempty"`
}

type ScrapedData struct {
	Type    string `json:"type"`
	Content string `json:"content"`
}

func main() {
	http.HandleFunc("/validateSelector", handleValidateSelector)
	http.HandleFunc("/scrape", handleScrape)
	http.Handle("/", http.FileServer(http.Dir("./static")))
	log.Fatal(http.ListenAndServe(":8080", nil))
}

func handleValidateSelector(w http.ResponseWriter, r *http.Request) {
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

	// Set headers
	for key, value := range req.Headers {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(key, value)
		})
	}

	var dataTypes []string

	// Add collector for the specified selector and detect available data types
	c.OnHTML(req.Selector, func(e *colly.HTMLElement) {
		e.ForEach(req.Selector, func(_ int, el *colly.HTMLElement) {
			dataTypes = append(dataTypes, el.Name)
		})
	})

	err := c.Visit(req.URL)
	if err != nil {
		responseError(w, "Failed to visit URL", http.StatusInternalServerError)
		return
	}

	// Deduplicate data types
	dataTypes = deduplicate(dataTypes)

	response := ValidateResponse{
		Success:   true,
		DataTypes: dataTypes,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func handleScrape(w http.ResponseWriter, r *http.Request) {
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

	// Set headers
	for key, value := range req.Headers {
		c.OnRequest(func(r *colly.Request) {
			r.Headers.Set(key, value)
		})
	}

	var scrapedData []ScrapedData

	// Add collector for the specified selector and content types
	c.OnHTML(req.Selector, func(e *colly.HTMLElement) {
		for _, dataType := range req.DataTypes {
			switch strings.ToLower(dataType) {
			case "text":
				scrapedData = append(scrapedData, ScrapedData{
					Type:    "Text",
					Content: e.Text,
				})
			case "title":
				scrapedData = append(scrapedData, ScrapedData{
					Type:    "Title",
					Content: e.ChildText("title"),
				})
			case "link":
				href := e.Request.AbsoluteURL(e.Attr("href"))
				scrapedData = append(scrapedData, ScrapedData{
					Type:    "Link",
					Content: href,
				})
			default:
				scrapedData = append(scrapedData, ScrapedData{
					Type:    "Unknown",
					Content: e.Text,
				})
			}
		}

		// Handle next page URL if provided
		if req.NextPageURL != "" {
			c.OnHTML("a[href]", func(e *colly.HTMLElement) {
				nextPageURL := e.Request.AbsoluteURL(e.Attr("href"))
				if strings.Contains(nextPageURL, req.NextPageURL) {
					err := c.Visit(nextPageURL)
					if err != nil {
						log.Println("Failed to visit next page:", err)
					}
				}
			})
		}
	})

	err := c.Visit(req.URL)
	if err != nil {
		responseError(w, "Failed to visit URL", http.StatusInternalServerError)
		return
	}

	response := ScrapeResponse{
		Data:        scrapedData,
		NextPageURL: req.NextPageURL,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func responseError(w http.ResponseWriter, message string, code int) {
	w.WriteHeader(code)
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"error": message})
}

func deduplicate(slice []string) []string {
	encountered := map[string]bool{}
	result := []string{}

	for v := range slice {
		if encountered[slice[v]] == true {
			// Do not add duplicate.
		} else {
			// Record this element as an encountered element.
			encountered[slice[v]] = true
			// Append to result slice.
			result = append(result, slice[v])
		}
	}
	// Return the new slice.
	return result
}

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type Product struct {
	ID           int    `json:"id"`
	Name         string  `json:"name"`
	Category     string  `json:"category"`
	Description  string  `json:"description"`
	Brand        string  `json:"brand"`
}

type SearchResponse struct {
	Products      []Product `json:"products"`
	TotalFound    int       `json:"total_found"`
	SearchTime    string    `json:"search_time"`
}

var store sync.Map

var brands = []string{"Alpha", "Beta", "Gamma", "Delta", "Epsilon", "Zeta", "Theta", "Omega"}
var categories = []string{"Electronics", "Books", "Home", "Garden", "Sports", "Toys", "Clothing", "Food"}
var descriptions = []string{
	"High quality product for everyday use",
	"Premium grade item with warrenty",
	"Budget friendly ooption with great reviews",
	"Top rated by customers worldwide",
	"Best seller in its category",
	"Eco-friendly and sustainable choice",
	"Professional grade equipment",
	"Limited edition collector item",
}

func generateProducts() {
	for i := 1; i <= 100000; i++ {
		brand := brands[i%len(brands)]
		p := Product{
			ID:              i,
			Name:            fmt.Sprintf("Product %s %d", brand, i),
			Category:        categories[i%len(categories)],
			Description:     descriptions[i%len(descriptions)],
			Brand:           brand,
		}
		store.Store(i, p)
	}
	log.Println("Generated 100,000 products")
}

func searchHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.ToLower(r.URL.Query().Get("q"))
	if q == "" {
		http.Error(w, `{"error": "query parameter 'q' is required"}`, http.StatusBadRequest)
		return
	}

	start := time.Now()
	var results []Product
	totalFound := 0
	checked := 0

	// check exactly 100 products then stop
	for i := 1; i <= 100000; i++ {
		if checked >= 100 {
			break
		}
		val, ok := store.Load(i)
		if !ok {
			continue
		}
		checked++ // count every product checked, not just matches

		p := val.(Product)
		name := strings.ToLower(p.Name)
		cat := strings.ToLower(p.Category)
		if strings.Contains(name, q) || strings.Contains(cat, q) {
			totalFound++
			if len(results) < 20 {
				results = append(results, p)
			}
		}
	}

	if results == nil {
		results = []Product{}
	}

	resp := SearchResponse{
		Products:     results,
		TotalFound:   totalFound,
		SearchTime:   time.Since(start).String(),
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status": "healthy"}`))
}

func main() {
	generateProducts()

	http.HandleFunc("/products/search", searchHandler)
	http.HandleFunc("/health", healthHandler)

	port := "8080"
	log.Printf("Server is running on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

// keep strconv import for potential use
var _ = strconv.Itoa
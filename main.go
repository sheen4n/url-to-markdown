package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
)

type ConversionRequest struct {
	URL string `json:"url"`
}

type ConversionResponse struct {
	Markdown string `json:"markdown"`
	Error    string `json:"error,omitempty"`
}

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	http.HandleFunc("/convert", handleConversion)
	http.HandleFunc("/health", handleHealth)

	log.Printf("Server starting on port %s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatal(err)
	}
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("healthy"))
}

func handleConversion(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req ConversionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondWithError(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		respondWithError(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Fetch the webpage
	resp, err := http.Get(req.URL)
	if err != nil {
		respondWithError(w, "Failed to fetch URL: "+err.Error(), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		respondWithError(w, "Failed to read response: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Convert to markdown
	converter := md.NewConverter("", true, nil)
	markdown, err := converter.ConvertString(string(body))
	if err != nil {
		respondWithError(w, "Failed to convert to markdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clean up the markdown
	markdown = strings.TrimSpace(markdown)

	response := ConversionResponse{
		Markdown: markdown,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func respondWithError(w http.ResponseWriter, message string, status int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(ConversionResponse{Error: message})
}

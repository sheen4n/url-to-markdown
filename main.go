package main

import (
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"

	md "github.com/JohannesKaufmann/html-to-markdown"
	goquery "github.com/PuerkitoBio/goquery"
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

	fs := http.FileServer(http.Dir("static"))
	http.Handle("/", fs)
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

func cleanupMarkdown(markdown string) string {
	// Remove multiple consecutive blank lines
	re := regexp.MustCompile(`\n{3,}`)
	markdown = re.ReplaceAllString(markdown, "\n\n")

	// Remove spaces at the end of lines
	re = regexp.MustCompile(`[ \t]+\n`)
	markdown = re.ReplaceAllString(markdown, "\n")

	// Ensure consistent newlines
	markdown = strings.ReplaceAll(markdown, "\r\n", "\n")

	// Remove extra spaces between words
	re = regexp.MustCompile(`[ \t]{2,}`)
	markdown = re.ReplaceAllString(markdown, " ")

	// Remove empty bullet points
	re = regexp.MustCompile(`(?m)^[-*+]\s*$\n`)
	markdown = re.ReplaceAllString(markdown, "")

	// Remove unnecessary line breaks between list items
	re = regexp.MustCompile(`\n\n([-*+]\s)`)
	markdown = re.ReplaceAllString(markdown, "\n$1")

	return strings.TrimSpace(markdown)
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

	// Configure converter with options
	converter := md.NewConverter("", true, &md.Options{
		StrongDelimiter: "**",
		EmDelimiter:     "_",
		LinkStyle:       "referenced",
		HeadingStyle:    "atx",
		// PreserveEmptyLines: false,
	})

	// Add custom rules
	converter.AddRules(md.Rule{
		Filter: []string{"br"},
		Replacement: func(content string, sel *goquery.Selection, opt *md.Options) *string {
			// Add a single newline for line breaks
			s := "\n"
			return &s
		},
	})

	markdown, err := converter.ConvertString(string(body))
	if err != nil {
		respondWithError(w, "Failed to convert to markdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Clean up the markdown
	markdown = cleanupMarkdown(markdown)

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

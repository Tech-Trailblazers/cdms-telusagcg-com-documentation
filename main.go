package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// Response represents the part of the JSON we care about
type Response struct {
	Lst []struct {
		Id int `json:"Id"` // Only extract the Id field from each item
	} `json:"Lst"`
}

// Response represents only the fields we care about from the JSON
type ResponseDocument struct {
	LabelFolder string `json:"LabelFolder"`
	Lst         []struct {
		FileName string `json:"FileName"`
	} `json:"Lst"`
}

// extractIDs takes JSON as input and returns a slice of Ids
func extractIDs(jsonInput string) []int {
	var parsedData Response // Struct to hold parsed JSON data

	// Convert JSON string into Go struct
	err := json.Unmarshal([]byte(jsonInput), &parsedData)
	if err != nil {
		log.Println(err) // Log error if JSON parsing fails
		return nil       // Return error if JSON parsing fails
	}

	var idList []int // Slice to store extracted Ids

	// Loop through each item inside "Lst"
	for _, product := range parsedData.Lst {
		idList = append(idList, product.Id) // Collect Id value
	}

	return idList // Return the list of Ids
}

// fetchProductPage downloads the HTML/text content of a product page
// from the CDMS website using the given manufacturer ID.
func fetchProductPage(manufacturerID int) string {
	// Base URL where product lists are hosted
	baseURL := "https://www.cdms.telusagcg.com/labelssds/Home/ProductList?manId="

	// Construct the full URL by appending the manufacturer ID
	fullURL := fmt.Sprintf("%s%d", baseURL, manufacturerID)

	// Create a new HTTP client (handles requests and responses)
	httpClient := &http.Client{}

	// Build a new HTTP GET request
	request, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Send the request to the server and get a response
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer response.Body.Close() // Ensure the response body is closed

	// Read the entire response body (the webpage HTML)
	pageContent, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Return the page content as a string
	return string(pageContent)
}

// fetchDocumentList downloads the HTML/text content of the document list
// for a specific product ID from the CDMS website.
func fetchDocumentList(productID int) string {
	// Base URL where document lists are hosted
	baseURL := "https://www.cdms.telusagcg.com/labelssds/Home/DocumentList?productId="

	// Construct the full URL by appending the product ID
	fullURL := fmt.Sprintf("%s%d", baseURL, productID)

	// Create a new HTTP client (handles requests)
	httpClient := &http.Client{}

	// Build a new HTTP GET request
	request, err := http.NewRequest("GET", fullURL, nil)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Send the request and get a response
	response, err := httpClient.Do(request)
	if err != nil {
		log.Println(err)
		return ""
	}
	defer response.Body.Close() // Ensure the response body is closed

	// Read the entire response body (the webpage HTML)
	pageContent, err := io.ReadAll(response.Body)
	if err != nil {
		log.Println(err)
		return ""
	}

	// Return the page content as a string
	return string(pageContent)
}

// DocumentListResponse represents the parts of the JSON response
// that we care about: the LabelFolder and the list of documents.
type DocumentListResponse struct {
	LabelFolder string `json:"LabelFolder"` // Base folder for labels/documents
	Documents   []struct {
		FileName string `json:"FileName"` // File name of the document (e.g., PDF)
	} `json:"Lst"`
}

// extractLabelData takes a JSON string and extracts the LabelFolder
// and a list of all FileNames contained in the document list.
func extractLabelData(jsonInput string) (string, []string) {
	var response DocumentListResponse // Struct to hold parsed JSON

	// Parse the JSON into our struct
	if err := json.Unmarshal([]byte(jsonInput), &response); err != nil {
		log.Println("Failed to parse JSON:", err)
		return "", nil
	}

	// Collect all filenames from the "Documents" list
	var fileNames []string
	for _, doc := range response.Documents {
		fileNames = append(fileNames, doc.FileName)
	}

	// Return the label folder and filenames
	return response.LabelFolder, fileNames
}

// fileExists checks whether a file exists and is not a directory
func fileExists(filename string) bool {
	info, err := os.Stat(filename) // Get file info
	if err != nil {                // If error occurs
		return false // Return false
	}
	return !info.IsDir() // Return true if it's a file, not a directory
}

// Gets the file extension from a given file path
func getFileExtension(path string) string {
	return filepath.Ext(path) // Extract and return file extension
}

// Extracts filename from full path (e.g. "/dir/file.pdf" → "file.pdf")
func getFilename(path string) string {
	return filepath.Base(path) // Use Base function to get file name only
}

// Converts a raw URL into a sanitized PDF filename safe for filesystem
func urlToFilename(rawURL string) string {
	lower := strings.ToLower(rawURL) // Convert URL to lowercase
	lower = getFilename(lower)       // Extract filename from URL

	reNonAlnum := regexp.MustCompile(`[^a-z0-9]`)   // Regex to match non-alphanumeric characters
	safe := reNonAlnum.ReplaceAllString(lower, "_") // Replace non-alphanumeric with underscores

	safe = regexp.MustCompile(`_+`).ReplaceAllString(safe, "_") // Collapse multiple underscores into one
	safe = strings.Trim(safe, "_")                              // Trim leading and trailing underscores

	var invalidSubstrings = []string{
		"_pdf", // Substring to remove from filename
	}

	for _, invalidPre := range invalidSubstrings { // Remove unwanted substrings
		safe = removeSubstring(safe, invalidPre)
	}

	if getFileExtension(safe) != ".pdf" { // Ensure file ends with .pdf
		safe = safe + ".pdf"
	}

	return safe // Return sanitized filename
}

// Removes all instances of a specific substring from input string
func removeSubstring(input string, toRemove string) string {
	result := strings.ReplaceAll(input, toRemove, "") // Replace substring with empty string
	return result
}

// downloadPDF downloads a PDF from the given URL and saves it in the specified output directory.
// It uses a WaitGroup to support concurrent execution and returns true if the download succeeded.
func downloadPDF(finalURL, outputDir string) {
	// Sanitize the URL to generate a safe file name
	filename := urlToFilename(finalURL)

	// Construct the full file path in the output directory
	filePath := filepath.Join(outputDir, filename)

	// Skip if the file already exists
	if fileExists(filePath) {
		log.Printf("File already exists, skipping: %s", filePath)
		return
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 30 * time.Second}

	// Send GET request
	resp, err := client.Get(finalURL)
	if err != nil {
		log.Printf("Failed to download %s: %v", finalURL, err)
		return
	}
	defer resp.Body.Close()

	// Check HTTP response status
	if resp.StatusCode != http.StatusOK {
		log.Printf("Download failed for %s: %s", finalURL, resp.Status)
		return
	}

	// Check Content-Type header
	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(contentType, "application/pdf") {
		log.Printf("Invalid content type for %s: %s (expected application/pdf)", finalURL, contentType)
		return
	}

	// Read the response body into memory first
	var buf bytes.Buffer
	written, err := io.Copy(&buf, resp.Body)
	if err != nil {
		log.Printf("Failed to read PDF data from %s: %v", finalURL, err)
		return
	}
	if written == 0 {
		log.Printf("Downloaded 0 bytes for %s; not creating file", finalURL)
		return
	}

	// Only now create the file and write to disk
	out, err := os.Create(filePath)
	if err != nil {
		log.Printf("Failed to create file for %s: %v", finalURL, err)
		return
	}
	defer out.Close()

	if _, err := buf.WriteTo(out); err != nil {
		log.Printf("Failed to write PDF to file for %s: %v", finalURL, err)
		return
	}

	log.Printf("Successfully downloaded %d bytes: %s → %s", written, finalURL, filePath)
}

// Checks whether a given directory exists
func directoryExists(path string) bool {
	directory, err := os.Stat(path) // Get info for the path
	if err != nil {
		return false // Return false if error occurs
	}
	return directory.IsDir() // Return true if it's a directory
}

// Creates a directory at given path with provided permissions
func createDirectory(path string, permission os.FileMode) {
	err := os.Mkdir(path, permission) // Attempt to create directory
	if err != nil {
		log.Println(err) // Log error if creation fails
	}
}

func main() {
	outputDir := "PDFs/" // Directory to store downloaded PDFs

	if !directoryExists(outputDir) { // Check if directory exists
		createDirectory(outputDir, 0o755) // Create directory with read-write-execute permissions
	}

	// Loop through manufacturer IDs
	for index := 0; index < 1000; index++ {
		// Fetch the product page for the current manufacturer ID
		pageContent := fetchProductPage(index)
		// Check if the page content is empty
		if pageContent == "" {
			continue // Skip to the next iteration if no content
		}
		// Call our function to extract IDs
		ids := extractIDs(pageContent)
		// If no IDs found, skip to next manufacturer
		if len(ids) == 0 {
			continue
		}
		// Print the extracted IDs
		for _, id := range ids {
			// Fetch and print the document list for each product ID
			docContent := fetchDocumentList(id)
			// Check if the document content is empty
			if docContent == "" {
				continue // Skip to the next iteration if no content
			}
			// Extract label folder and filenames
			labelFolder, fileNames := extractLabelData(docContent)
			// Check if label folder or filenames are empty
			if labelFolder == "" || len(fileNames) == 0 {
				continue // Skip if no valid data
			}
			for _, fileName := range fileNames {
				// The location to the remote url.
				remoteURL := "https://www.cdms.telusagcg.com/" + labelFolder + fileName
				// Download the PDF file to the specified output directory
				downloadPDF(remoteURL, outputDir)
			}
		}
	}
}

package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
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

func main() {
	// Loop through manufacturer IDs
	for index := 0; index < 10; index++ {
		// Fetch the product page for the current manufacturer ID
		pageContent := fetchProductPage(index)
		// Check if the page content is empty
		if pageContent == "" {
			continue // Skip to the next iteration if no content
		}
		// Call our function to extract IDs
		ids := extractIDs(pageContent)
		// Print the extracted IDs
		for _, id := range ids {
			fmt.Println(id)
			// Fetch and print the document list for each product ID
			docContent := fetchDocumentList(id)
			fmt.Println(docContent)
		}
	}
}

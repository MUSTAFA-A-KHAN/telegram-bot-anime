package model

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"time"
)

// Car represents the structure of a car brand and its models
type Car struct {
	Brand  string   `json:"brand"`
	Models []string `json:"models"`
}

// GetRandomWord fetches a random word from the provided API
func GetRandomCar() (string, error) {
	rand.Seed(time.Now().UnixNano())

	// API URL
	apiURL := "https://raw.githubusercontent.com/matthlavacka/car-list/refs/heads/master/car-list.json"

	// Fetch data from the API
	response, err := http.Get(apiURL)
	if err != nil {
		log.Fatalf("Error fetching data from API: %v", err)
	}
	defer response.Body.Close()

	// Parse the JSON response
	var cars []Car
	err = json.NewDecoder(response.Body).Decode(&cars)
	if err != nil {
		log.Fatalf("Error decoding JSON: %v", err)
	}

	// Select a random car brand and model
	randomBrandIndex := rand.Intn(len(cars))
	randomBrand := cars[randomBrandIndex]

	randomModelIndex := rand.Intn(len(randomBrand.Models))
	randomModel := randomBrand.Models[randomModelIndex]

	// Print the random car brand and model
	fmt.Printf("Random Car: %s %s\n", randomBrand.Brand, randomModel)
	return randomBrand.Brand, nil
}

package model

import (
	"encoding/json"
	"io"
	"math/rand"
	"net/http"
	"time"
)

type Flower struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func GetRandomFlower() (string, error) {
	apiUrl := "https://raw.githubusercontent.com/MUSTAFA-A-KHAN/json-data-hub/refs/heads/main/500_flowers.json"

	response, err := http.Get(apiUrl)
	if err != nil {
		return "", err
	}
	var flowers []Flower
	bodybytes, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}
	json.Unmarshal(bodybytes, &flowers)
	rand.Seed(time.Now().UnixNano())
	randomFlower := flowers[rand.Intn(500)]

	return randomFlower.Name, nil
}

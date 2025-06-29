package service

import (
	"encoding/json"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

// ExportAllData fetches all data from MongoDB and returns it as JSON bytes
func ExportAllData(client *mongo.Client, collection string) ([]byte, error) {
	data := repository.ReadAllDoc(client, collection)
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, err
	}
	return jsonData, nil
}

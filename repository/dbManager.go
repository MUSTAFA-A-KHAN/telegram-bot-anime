package repository

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DbManager() *mongo.Client {
	fmt.Print("into DBmanager")
	passsword := "pass@123"
	encodedPassword := url.QueryEscape(passsword)
	clientOptions := options.Client().ApplyURI("mongodb+srv://Mkhan62608gmailcom:" + encodedPassword + "@cluster0.zuzzadg.mongodb.net/?retryWrites=true&w=majority")

	// Connect to MongoDB
	client, err := mongo.Connect(context.TODO(), clientOptions)
	if err != nil {
		log.Fatal(err)
	}

	// Ensure connection is established
	err = client.Ping(context.TODO(), nil)
	if err != nil {
		return nil
	}
	fmt.Println("Connected to MongoDB successfully!")
	return client
}
func InsertDoc(ID int, Name string, chatID int64, client *mongo.Client, collection string) {
	// Select the database and collections
	database := client.Database("Telegram")
	// movieCollection := database.Collection("CrocEn")
	commentCollection := database.Collection(collection)

	// // Example: Find a movie document and get the ObjectId
	// var movie bson.M
	// err = movieCollection.FindOne(context.TODO(), bson.D{}).Decode(&movie)
	// if err != nil {
	// 	log.Fatal(err)
	// }
	// fmt.Println(movie["title"])
	// movieID := movie["_id"].(primitive.ObjectID)
	// fmt.Println("timestamp", movieID.Timestamp(), "\n", movieID.String())

	// Create the comment document with the movie's ObjectId
	comment := bson.D{
		{Key: "ID", Value: ID},
		{Key: "Name", Value: Name},
		{Key: "chat_ID", Value: chatID}, // Pass the ObjectId here
	}

	// Insert the comment into the NewCollection
	insertResult, err := commentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println("Inserted comment with ID:", insertResult.InsertedID)
}
func ReadAllDoc(client *mongo.Client) []bson.M {
	database := client.Database("Telegram")
	// movieCollection := database.Collection("CrocEn")
	commentCollection := database.Collection("CrocEn")
	// Optionally, print all comments from the collection
	cursor, err := commentCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())
	var results []bson.M
	for cursor.Next(context.TODO()) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			log.Fatal(err)
		}
		fmt.Println(result)
	}

	// Close the MongoDB client connection
	err = client.Disconnect(context.TODO())
	if err != nil {
		log.Fatal(err)
	}
	return results

}

// Function to count occurrences of each ID along with the Name
func CountIDOccurrences(client *mongo.Client, collection string) ([]map[string]interface{}, error) {
	database := client.Database("Telegram")
	commentCollection := database.Collection(collection)

	// Aggregation pipeline to count occurrences of each ID and include the Name
	pipeline := mongo.Pipeline{
		// Group by ID, count occurrences, and include Name
		{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},                                    // Group by the "ID" field
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},        // Count occurrences
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}}, // Get the first "Name" encountered for the grouped ID
		}}},

		// Sort by count (descending)
		{{"$sort", bson.D{{Key: "count", Value: -1}}}},
	}

	// Execute the aggregation query
	cursor, err := commentCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	var results []map[string]interface{}
	for cursor.Next(context.TODO()) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	// Check for any errors that occurred during iteration
	if err := cursor.Err(); err != nil {
		return nil, err
	}

	return results, nil
}
func InsertUserInfo(userInfo model.UserInfo, client *mongo.Client) {
	database := client.Database("Telegram")
	// movieCollection := database.Collection("CrocEn")
	commentCollection := database.Collection("UserInfo")
	commentCollection.InsertOne(context.TODO(), userInfo)
}

// GetUserStatsByID returns the count and name for a specific user ID from the given collection
func GetUserStatsByID(client *mongo.Client, collection string, userID int) (map[string]interface{}, error) {
	database := client.Database("Telegram")
	commentCollection := database.Collection(collection)

	// Aggregation pipeline to match the userID and count occurrences
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{Key: "ID", Value: userID}}}},
		{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}},
	}

	cursor, err := commentCollection.Aggregate(context.TODO(), pipeline)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if cursor.Next(context.TODO()) {
		var result map[string]interface{}
		if err := cursor.Decode(&result); err != nil {
			return nil, err
		}
		return result, nil
	}

	return nil, fmt.Errorf("no stats found for user ID %d", userID)
}

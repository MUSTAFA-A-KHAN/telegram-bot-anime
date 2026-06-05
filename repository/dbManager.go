package repository

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"sync"

	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var (
	clientInstance *mongo.Client
	clientOnce     sync.Once
)

func DbManager() *mongo.Client {
	clientOnce.Do(func() {
		fmt.Print("into DBmanager")
		passsword := "pass@123"
		encodedPassword := url.QueryEscape(passsword)
		clientOptions := options.Client().ApplyURI("mongodb+srv://Mkhan62608gmailcom:" + encodedPassword + "@cluster0.zuzzadg.mongodb.net/?retryWrites=true&w=majority")

		// Connect to MongoDB
		client, err := mongo.Connect(context.TODO(), clientOptions)
		if err != nil {
			log.Print(err)
			return
		}

		// Ensure connection is established
		err = client.Ping(context.TODO(), nil)
		if err != nil {
			return
		}
		fmt.Println("Connected to MongoDB successfully!")
		clientInstance = client
	})

	return clientInstance
}
func InsertDoc(ID int, Name string, chatID int64, client *mongo.Client, collection string) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in InsertDoc: %v", r)
		}
	}()

	if client == nil {
		log.Println("MongoDB client is nil in InsertDoc, skipping insert")
		return
	}

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
		log.Println("Error inserting document in InsertDoc:", err)
		return
	}
	fmt.Println("Inserted comment with ID:", insertResult.InsertedID)
}

func InsertWordleBonusDoc(ID int, Name string, chatID int64, client *mongo.Client, collection string, points int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in InsertWordleBonusDoc: %v", r)
		}
	}()

	if client == nil {
		log.Println("MongoDB client is nil in InsertWordleBonusDoc, skipping insert")
		return
	}

	database := client.Database("Telegram")
	commentCollection := database.Collection(collection)

	comment := bson.D{
		{Key: "ID", Value: ID},
		{Key: "Name", Value: Name},
		{Key: "chat_ID", Value: chatID},
		{Key: "Points", Value: points},
	}

	insertResult, err := commentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		log.Println("Error inserting document in InsertWordleBonusDoc:", err)
		return
	}
	fmt.Println("Inserted Wordle bonus comment with ID:", insertResult.InsertedID, "Points:", points)
}

func InsertWordleDoc(ID int, Name string, chatID int64, client *mongo.Client, collection string, attempts int) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in InsertWordleDoc: %v", r)
		}
	}()

	if client == nil {
		log.Println("MongoDB client is nil in InsertWordleDoc, skipping insert")
		return
	}

	database := client.Database("Telegram")
	commentCollection := database.Collection(collection)

	points := 25 - attempts + 1
	if points < 1 {
		points = 1 // Ensure minimum 1 point for winning
	}

	comment := bson.D{
		{Key: "ID", Value: ID},
		{Key: "Name", Value: Name},
		{Key: "chat_ID", Value: chatID},
		{Key: "Points", Value: points},
	}

	insertResult, err := commentCollection.InsertOne(context.TODO(), comment)
	if err != nil {
		log.Println("Error inserting document in InsertWordleDoc:", err)
		return
	}
	fmt.Println("Inserted Wordle comment with ID:", insertResult.InsertedID, "Points:", points)
}

func ReadAllDoc(client *mongo.Client, collection string) []bson.M {
	database := client.Database("Telegram")
	// movieCollection := database.Collection("CrocEn")
	commentCollection := database.Collection(collection)
	// Optionally, print all comments from the collection
	cursor, err := commentCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Println(err)
	}
	defer cursor.Close(context.TODO())
	var results []bson.M
	for cursor.Next(context.TODO()) {
		var result bson.M
		if err := cursor.Decode(&result); err != nil {
			log.Print(err)
		}
		fmt.Println(result)
		results = append(results, result)
	}
	return results

}

// Function to count occurrences of each ID along with the Name
func CountIDOccurrences(client *mongo.Client, collection string, chatID int64) ([]map[string]interface{}, error) {
	database := client.Database("Telegram")
	commentCollection := database.Collection(collection)

	// Aggregation pipeline to count occurrences of each ID and include the Name
	var pipeline mongo.Pipeline

	// If chatID is provided (non-zero), filter by chat_ID
	if chatID != 0 {
		pipeline = append(pipeline, bson.D{{"$match", bson.D{{Key: "chat_ID", Value: chatID}}}})
	}

	var groupStage bson.D
	if collection == "WordleEn" {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$Points", 25}}}}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	} else if collection == "ScramyEn" {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: "$Points"}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	} else {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	}

	pipeline = append(pipeline,
		// Group by ID, count occurrences (or points), and include Name
		groupStage,
		// Sort by count (descending)
		bson.D{{"$sort", bson.D{{Key: "count", Value: -1}}}},
	)

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

	var groupStage bson.D
	if collection == "WordleEn" {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: bson.D{{Key: "$ifNull", Value: bson.A{"$Points", 25}}}}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	} else if collection == "ScramyEn" {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: "$Points"}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	} else {
		groupStage = bson.D{{"$group", bson.D{
			{Key: "_id", Value: "$ID"},
			{Key: "count", Value: bson.D{{Key: "$sum", Value: 1}}},
			{Key: "Name", Value: bson.D{{Key: "$first", Value: "$Name"}}},
		}}}
	}

	// Aggregation pipeline to match the userID and count occurrences (or points)
	pipeline := mongo.Pipeline{
		{{"$match", bson.D{{Key: "ID", Value: userID}}}},
		groupStage,
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

// HasFreeEordle checks if the user has a free Eordle available for today.
func HasFreeEordle(client *mongo.Client, userID int) bool {
	if client == nil {
		log.Println("MongoDB client is nil in HasFreeEordle")
		return false
	}

	database := client.Database("Telegram")
	collection := database.Collection("EordleUsage")

	today := time.Now().Format("2006-01-02")

	var result bson.M
	err := collection.FindOne(context.TODO(), bson.D{
		{Key: "ID", Value: userID},
		{Key: "Date", Value: today},
	}).Decode(&result)

	if err != nil {
		if err == mongo.ErrNoDocuments {
			return true // No usage for today, free eordle is available
		}
		log.Printf("Error checking free Eordle: %v", err)
		return false
	}

	return false // Used already
}

// UseFreeEordle marks the free Eordle as used for today for a specific user.
func UseFreeEordle(client *mongo.Client, userID int) {
	if client == nil {
		log.Println("MongoDB client is nil in UseFreeEordle")
		return
	}

	database := client.Database("Telegram")
	collection := database.Collection("EordleUsage")

	today := time.Now().Format("2006-01-02")

	doc := bson.D{
		{Key: "ID", Value: userID},
		{Key: "Date", Value: today},
	}

	_, err := collection.InsertOne(context.TODO(), doc)
	if err != nil {
		log.Printf("Error inserting Eordle usage: %v", err)
	}
}

// GetCurrentPoints gets the current points of a user from WordleEn collection
func GetCurrentPoints(client *mongo.Client, userID int) int {
	stats, err := GetUserStatsByID(client, "WordleEn", userID)
	if err != nil {
		return 0
	}

	if val, ok := stats["count"]; ok {
		switch v := val.(type) {
		case int32:
			return int(v)
		case int64:
			return int(v)
		case int:
			return v
		case float64:
			return int(v)
		}
	}
	return 0
}

// DeductWordlePoints deducts a given amount of points from a user
func DeductWordlePoints(client *mongo.Client, userID int, name string, chatID int64, points int) {
	InsertWordleBonusDoc(userID, name, chatID, client, "WordleEn", -points)
}

// GetEquippedEmojis returns the list of emojis equipped by a user
func GetEquippedEmojis(client *mongo.Client, userID int) ([]string, error) {
	database := client.Database("Telegram")
	collection := database.Collection("UserEmojis")

	filter := bson.M{"UserID": userID}
	var result struct {
		EquippedEmojis []string `bson:"EquippedEmojis"`
	}

	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, err
	}
	return result.EquippedEmojis, nil
}

// GetPurchasedEmojis returns the list of emojis purchased by a user
func GetPurchasedEmojis(client *mongo.Client, userID int) ([]string, error) {
	database := client.Database("Telegram")
	collection := database.Collection("UserEmojis")

	filter := bson.M{"UserID": userID}
	var result struct {
		PurchasedEmojis []string `bson:"PurchasedEmojis"`
	}

	err := collection.FindOne(context.TODO(), filter).Decode(&result)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return []string{}, nil
		}
		return nil, err
	}
	return result.PurchasedEmojis, nil
}

// PurchaseEmoji adds an emoji to the user's purchased list
func PurchaseEmoji(client *mongo.Client, userID int, emoji string) error {
	database := client.Database("Telegram")
	collection := database.Collection("UserEmojis")

	filter := bson.M{"UserID": userID}
	update := bson.M{
		"$addToSet": bson.M{"PurchasedEmojis": emoji},
	}

	opts := options.Update().SetUpsert(true)
	_, err := collection.UpdateOne(context.TODO(), filter, update, opts)
	return err
}

// ToggleEquipEmoji toggles whether an emoji is equipped for a user
func ToggleEquipEmoji(client *mongo.Client, userID int, emoji string) (bool, error) {
	database := client.Database("Telegram")
	collection := database.Collection("UserEmojis")

	filter := bson.M{"UserID": userID}
	var result struct {
		EquippedEmojis []string `bson:"EquippedEmojis"`
	}
	err := collection.FindOne(context.TODO(), filter).Decode(&result)

	isEquipped := false
	if err == nil {
		for _, e := range result.EquippedEmojis {
			if e == emoji {
				isEquipped = true
				break
			}
		}
	} else if err != mongo.ErrNoDocuments {
		return false, err
	}

	var update bson.M
	if isEquipped {
		update = bson.M{"$pull": bson.M{"EquippedEmojis": emoji}}
	} else {
		update = bson.M{"$addToSet": bson.M{"EquippedEmojis": emoji}}
	}

	opts := options.Update().SetUpsert(true)
	_, err = collection.UpdateOne(context.TODO(), filter, update, opts)
	if err != nil {
		return false, err
	}

	return !isEquipped, nil
}

package repository

import (
	"context"
	"fmt"
	"log"
	"net/url"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func DbManager(ID int, Name string, chatID int64) {
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
		log.Fatal(err)
	}
	fmt.Println("Connected to MongoDB successfully!")

	// Select the database and collections
	database := client.Database("Telegram")
	// movieCollection := database.Collection("CrocEn")
	commentCollection := database.Collection("CrocEn")

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

	// Optionally, print all comments from the collection
	cursor, err := commentCollection.Find(context.TODO(), bson.D{})
	if err != nil {
		log.Fatal(err)
	}
	defer cursor.Close(context.TODO())

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

}

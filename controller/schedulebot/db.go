package schedulebot

import (
	"context"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/config"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ScheduledMessage struct {
	ID             primitive.ObjectID `bson:"_id,omitempty"`
	ChatID         int64              `bson:"chat_id"`
	AddedBy        int                `bson:"added_by"`
	CronExpression string             `bson:"cron_expression"`

	// Message components
	Text      string `bson:"text,omitempty"`
	Caption   string `bson:"caption,omitempty"`
	FileID    string `bson:"file_id,omitempty"`
	MediaType string `bson:"media_type,omitempty"` // photo, audio, document, video, animation, voice
}

func GetCollection(client *mongo.Client) *mongo.Collection {
	return client.Database(config.App.DatabaseName).Collection("ScheduledMessages")
}

func saveSchedule(client *mongo.Client, msg ScheduledMessage) (*mongo.InsertOneResult, error) {
	collection := GetCollection(client)
	return collection.InsertOne(context.Background(), msg)
}

func deleteSchedule(client *mongo.Client, chatID int64, id primitive.ObjectID) (*mongo.DeleteResult, error) {
	collection := GetCollection(client)
	return collection.DeleteOne(context.Background(), bson.M{"_id": id, "chat_id": chatID})
}

func getSchedules(client *mongo.Client, chatID int64) ([]ScheduledMessage, error) {
	collection := GetCollection(client)
	cursor, err := collection.Find(context.Background(), bson.M{"chat_id": chatID})
	if err != nil {
		return nil, err
	}
	var schedules []ScheduledMessage
	if err := cursor.All(context.Background(), &schedules); err != nil {
		return nil, err
	}
	return schedules, nil
}

func getAllSchedules(client *mongo.Client) ([]ScheduledMessage, error) {
	collection := GetCollection(client)
	cursor, err := collection.Find(context.Background(), bson.M{})
	if err != nil {
		return nil, err
	}
	var schedules []ScheduledMessage
	if err := cursor.All(context.Background(), &schedules); err != nil {
		return nil, err
	}
	return schedules, nil
}

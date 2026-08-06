package repository

import (
	"context"
	"time"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/config"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const userProfilesCollection = "UserProfiles"

// GetUserProfile fetches a user profile by ID, creating one if it doesn't exist.
func GetUserProfile(client *mongo.Client, userID int64, username string) (*model.UserProfile, error) {
	collection := client.Database(config.App.DatabaseName).Collection(userProfilesCollection)

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$setOnInsert": bson.M{
			"user_id": userID,
			// "username":              username,
			"level":                 1,
			"xp":                    0,
			"coins":                 0,
			"games_played":          0,
			"wins":                  0,
			"current_streak":        0,
			"active_days_this_week": []string{},
			"created_at":            time.Now(),
		},
		"$set": bson.M{
			"username":   username, // always update username to latest
			"updated_at": time.Now(),
		},
	}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var profile model.UserProfile
	err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&profile)
	if err != nil {
		return nil, err
	}

	return &profile, nil
}

// UpdateUserProfile updates an existing user profile.
func UpdateUserProfile(client *mongo.Client, profile *model.UserProfile) error {
	collection := client.Database(config.App.DatabaseName).Collection(userProfilesCollection)

	profile.UpdatedAt = time.Now()
	filter := bson.M{"user_id": profile.UserID}
	update := bson.M{"$set": profile} // Replace the entire document fields

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	return err
}

// UpdateUserProfileFields updates specific fields of a user profile.
func UpdateUserProfileFields(client *mongo.Client, userID int64, updateFields bson.M) error {
	collection := client.Database(config.App.DatabaseName).Collection(userProfilesCollection)

	updateFields["updated_at"] = time.Now()
	filter := bson.M{"user_id": userID}
	update := bson.M{"$set": updateFields}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	return err
}

// IncrementUserProfileStats increments numerical fields like XP, Coins, etc.
func IncrementUserProfileStats(client *mongo.Client, userID int64, incFields bson.M) error {
	collection := client.Database(config.App.DatabaseName).Collection(userProfilesCollection)

	filter := bson.M{"user_id": userID}
	update := bson.M{
		"$inc": incFields,
		"$set": bson.M{"updated_at": time.Now()},
	}

	_, err := collection.UpdateOne(context.TODO(), filter, update)
	return err
}

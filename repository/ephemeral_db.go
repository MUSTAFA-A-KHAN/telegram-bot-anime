package repository

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const ephemeralDatabase = "Telegram"

// ---- EphemeralRequests ----
// Tracks the pending /ephemeral prompt each user has open, so a following reply
// can be routed as a whisper. Replaces the in-memory ephemeralRequests map.

type EphemeralRequestDoc struct {
	UserID         int64 `bson:"_id"`
	ChatID         int64 `bson:"chat_id"`
	EphemeralMsgID int   `bson:"ephemeral_msg_id"`
}

func UpsertEphemeralRequest(client *mongo.Client, userID int64, chatID int64, ephemeralMsgID int) {
	if client == nil {
		return
	}
	coll := client.Database(ephemeralDatabase).Collection("EphemeralRequests")
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"chat_id": chatID, "ephemeral_msg_id": ephemeralMsgID}},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("UpsertEphemeralRequest failed for user %d: %v", userID, err)
	}
}

// TakeEphemeralRequest atomically reads and removes the pending request for a
// user (find-and-delete), mirroring the old lock + delete semantics.
func TakeEphemeralRequest(client *mongo.Client, userID int64) (*EphemeralRequestDoc, error) {
	if client == nil {
		return nil, fmt.Errorf("nil mongo client")
	}
	coll := client.Database(ephemeralDatabase).Collection("EphemeralRequests")
	var doc EphemeralRequestDoc
	err := coll.FindOneAndDelete(context.TODO(), bson.M{"_id": userID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// ---- KnownUserIDs ----
// Maps lowercase usernames to user IDs. Replaces the in-memory knownUserIDs map.

type KnownUserDoc struct {
	Username string `bson:"_id"`
	UserID   int64  `bson:"user_id"`
}

func UpsertKnownUser(client *mongo.Client, username string, userID int64) {
	if client == nil || username == "" {
		return
	}
	coll := client.Database(ephemeralDatabase).Collection("KnownUserIDs")
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": username},
		bson.M{"$set": bson.M{"user_id": userID}},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("UpsertKnownUser failed for %s: %v", username, err)
	}
}

func GetKnownUser(client *mongo.Client, username string) (int64, bool) {
	if client == nil || username == "" {
		return 0, false
	}
	coll := client.Database(ephemeralDatabase).Collection("KnownUserIDs")
	var doc KnownUserDoc
	err := coll.FindOne(context.TODO(), bson.M{"_id": username}).Decode(&doc)
	if err != nil {
		return 0, false
	}
	return doc.UserID, true
}

// ---- StoredWhispers ----
// Whispers that could not be delivered as ephemeral messages, retrieved later
// via /ephemeral. One document per recipient holding an array of whispers.
// Replaces the in-memory storedWhispers map.

type StoredWhisperDoc struct {
	ChatID            int64  `bson:"chat_id"`
	MediaType         string `bson:"media_type"`
	FileID            string `bson:"file_id"`
	MediaLength       int    `bson:"media_length"`
	Caption           string `bson:"caption"`
	Text              string `bson:"text"`
	SenderID          int64  `bson:"sender_id"`
	SenderUsername    string `bson:"sender_username"`
	SenderName        string `bson:"sender_name"`
	SenderEphemeralID int64  `bson:"sender_ephemeral_id"`
}

type StoredWhispersDoc struct {
	UserID   int64              `bson:"_id"`
	Whispers []StoredWhisperDoc `bson:"whispers"`
}

func AddStoredWhisper(client *mongo.Client, userID int64, w StoredWhisperDoc) {
	if client == nil {
		return
	}
	coll := client.Database(ephemeralDatabase).Collection("StoredWhispers")
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$push": bson.M{"whispers": w}},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("AddStoredWhisper failed for user %d: %v", userID, err)
	}
}

// TakeStoredWhispers atomically reads and removes all stored whispers for a user.
func TakeStoredWhispers(client *mongo.Client, userID int64) []StoredWhisperDoc {
	if client == nil {
		return nil
	}
	coll := client.Database(ephemeralDatabase).Collection("StoredWhispers")
	var doc StoredWhispersDoc
	err := coll.FindOneAndDelete(context.TODO(), bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		return nil
	}
	return doc.Whispers
}

// ---- WhisperEphemeralSources ----
// Maps a delivered ephemeral message id back to the original whisper sender, so
// a direct reply can be forwarded back. Replaces whisperEphemeralSources map.

type WhisperSourceDoc struct {
	EphemeralMsgID    int64  `bson:"_id"`
	SenderID          int64  `bson:"sender_id"`
	SenderUsername    string `bson:"sender_username"`
	SenderName        string `bson:"sender_name"`
	SenderEphemeralID int64  `bson:"sender_ephemeral_id"`
}

func UpsertWhisperSource(client *mongo.Client, src WhisperSourceDoc) {
	if client == nil {
		return
	}
	coll := client.Database(ephemeralDatabase).Collection("WhisperEphemeralSources")
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": src.EphemeralMsgID},
		bson.M{"$set": bson.M{
			"sender_id":           src.SenderID,
			"sender_username":     src.SenderUsername,
			"sender_name":         src.SenderName,
			"sender_ephemeral_id": src.SenderEphemeralID,
		}},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("UpsertWhisperSource failed for ephemeral %d: %v", src.EphemeralMsgID, err)
	}
}

// TakeWhisperSource atomically reads and removes the source for an ephemeral id.
func TakeWhisperSource(client *mongo.Client, ephemeralMsgID int64) (*WhisperSourceDoc, error) {
	if client == nil {
		return nil, fmt.Errorf("nil mongo client")
	}
	coll := client.Database(ephemeralDatabase).Collection("WhisperEphemeralSources")
	var doc WhisperSourceDoc
	err := coll.FindOneAndDelete(context.TODO(), bson.M{"_id": ephemeralMsgID}).Decode(&doc)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &doc, nil
}

// ---- LatestUserEphemeral ----
// Tracks each user's currently open ephemeral thread (most recent ephemeral id).
// Replaces the in-memory latestUserEphemeral map.

type LatestEphemeralDoc struct {
	UserID         int64 `bson:"_id"`
	EphemeralMsgID int64 `bson:"ephemeral_msg_id"`
}

func SetLatestUserEphemeral(client *mongo.Client, userID int64, ephemeralMsgID int64) {
	if client == nil || userID == 0 || ephemeralMsgID == 0 {
		return
	}
	coll := client.Database(ephemeralDatabase).Collection("LatestUserEphemeral")
	_, err := coll.UpdateOne(context.TODO(),
		bson.M{"_id": userID},
		bson.M{"$set": bson.M{"ephemeral_msg_id": ephemeralMsgID}},
		options.Update().SetUpsert(true))
	if err != nil {
		log.Printf("SetLatestUserEphemeral failed for user %d: %v", userID, err)
	}
}

func GetLatestUserEphemeral(client *mongo.Client, userID int64) int64 {
	if client == nil {
		return 0
	}
	coll := client.Database(ephemeralDatabase).Collection("LatestUserEphemeral")
	var doc LatestEphemeralDoc
	err := coll.FindOne(context.TODO(), bson.M{"_id": userID}).Decode(&doc)
	if err != nil {
		return 0
	}
	return doc.EphemeralMsgID
}

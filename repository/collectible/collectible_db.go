package collectible

import (
	"context"
	"errors"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model/collectible"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	dbName              = "Telegram"
	templatesCollection = "CollectibleTemplates"
	itemsCollection     = "Collectibles"
	listingsCollection  = "MarketListings"
	counterCollection   = "Counters" // Used for generating unique serial numbers
)

// GetTemplates returns all available collectible templates
func GetTemplates(client *mongo.Client) ([]collectible.Template, error) {
	collection := client.Database(dbName).Collection(templatesCollection)
	var templates []collectible.Template
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &templates); err != nil {
		return nil, err
	}
	return templates, nil
}

// BootstrapTemplates seeds the database with initial templates if empty
func BootstrapTemplates(client *mongo.Client) error {
	templates, err := GetTemplates(client)
	if err != nil {
		return err
	}
	if len(templates) > 0 {
		return nil // Already seeded
	}

	collection := client.Database(dbName).Collection(templatesCollection)
	initialTemplates := []interface{}{
		collectible.Template{ID: primitive.NewObjectID().Hex(), Name: "Naruto Run", Rarity: collectible.RarityCommon, Emoji: "🏃", ImageURL: "https://encrypted-tbn0.gstatic.com/images?q=tbn:ANd9GcTBq1hBIt2Ve2P6TfmCehLeuNvVEubCxTRphQ&s"},
		collectible.Template{ID: primitive.NewObjectID().Hex(), Name: "Goku Hair", Rarity: collectible.RarityUncommon, Emoji: "🔥"},
		collectible.Template{ID: primitive.NewObjectID().Hex(), Name: "Sakura Shrine", Rarity: collectible.RarityRare, Emoji: "🌸"},
		collectible.Template{ID: primitive.NewObjectID().Hex(), Name: "Gojo Eyes", Rarity: collectible.RarityEpic, Emoji: "🌌"},
		collectible.Template{ID: primitive.NewObjectID().Hex(), Name: "Levi Sword", Rarity: collectible.RarityLegendary, Emoji: "⚔️"},
	}

	_, err = collection.InsertMany(context.TODO(), initialTemplates)
	return err
}

// GetNextSerialNumber returns the next available serial number for a given template ID
func GetNextSerialNumber(client *mongo.Client, templateID string) (int, error) {
	collection := client.Database(dbName).Collection(counterCollection)

	filter := bson.M{"_id": "serial_" + templateID}
	update := bson.M{"$inc": bson.M{"seq": 1}}
	opts := options.FindOneAndUpdate().SetUpsert(true).SetReturnDocument(options.After)

	var result struct {
		Seq int `bson:"seq"`
	}
	err := collection.FindOneAndUpdate(context.TODO(), filter, update, opts).Decode(&result)
	if err != nil {
		return 0, err
	}
	return result.Seq, nil
}

// MintItem creates a new collectible item for a user
func MintItem(client *mongo.Client, item collectible.Item) (collectible.Item, error) {
	collection := client.Database(dbName).Collection(itemsCollection)

	if item.ID == "" {
		item.ID = primitive.NewObjectID().Hex()
	}

	_, err := collection.InsertOne(context.TODO(), item)
	return item, err
}

// GetUserInventory returns all items owned by a specific user
func GetUserInventory(client *mongo.Client, userID int) ([]collectible.Item, error) {
	collection := client.Database(dbName).Collection(itemsCollection)
	var items []collectible.Item
	cursor, err := collection.Find(context.TODO(), bson.M{"owner_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &items); err != nil {
		return nil, err
	}
	return items, nil
}

// GetItemByID returns a specific item by its ID
func GetItemByID(client *mongo.Client, itemID string) (collectible.Item, error) {
	collection := client.Database(dbName).Collection(itemsCollection)
	var item collectible.Item
	err := collection.FindOne(context.TODO(), bson.M{"_id": itemID}).Decode(&item)
	return item, err
}

// CreateListing creates a new marketplace listing
func CreateListing(client *mongo.Client, listing collectible.MarketListing) error {
	collection := client.Database(dbName).Collection(listingsCollection)

	// Prevent duplicate listings for the same item
	count, err := collection.CountDocuments(context.TODO(), bson.M{"item_id": listing.ItemID})
	if err != nil {
		return err
	}
	if count > 0 {
		return errors.New("item is already listed on the marketplace")
	}

	if listing.ID == "" {
		listing.ID = primitive.NewObjectID().Hex()
	}

	_, err = collection.InsertOne(context.TODO(), listing)
	return err
}

// GetListings returns all active marketplace listings
func GetListings(client *mongo.Client) ([]collectible.MarketListing, error) {
	collection := client.Database(dbName).Collection(listingsCollection)
	var listings []collectible.MarketListing
	cursor, err := collection.Find(context.TODO(), bson.M{})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(context.TODO())

	if err := cursor.All(context.TODO(), &listings); err != nil {
		return nil, err
	}
	return listings, nil
}

// GetListingByID returns a specific listing
func GetListingByID(client *mongo.Client, listingID string) (collectible.MarketListing, error) {
	collection := client.Database(dbName).Collection(listingsCollection)
	var listing collectible.MarketListing
	err := collection.FindOne(context.TODO(), bson.M{"_id": listingID}).Decode(&listing)
	return listing, err
}

// ProcessPurchase transfers ownership and removes the listing atomically (in concept)
func ProcessPurchase(client *mongo.Client, listingID string, itemID string, buyerID int) error {
	// In MongoDB standalone, we do this sequentially. In replica sets, we could use transactions.

	// 1. Delete listing
	listingsColl := client.Database(dbName).Collection(listingsCollection)
	res, err := listingsColl.DeleteOne(context.TODO(), bson.M{"_id": listingID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return errors.New("listing not found or already purchased")
	}

	// 2. Transfer ownership
	itemsColl := client.Database(dbName).Collection(itemsCollection)
	_, err = itemsColl.UpdateOne(
		context.TODO(),
		bson.M{"_id": itemID},
		bson.M{"$set": bson.M{"owner_id": buyerID}},
	)
	return err
}

// DeleteListing completely removes a listing (e.g., when a user cancels it)
func DeleteListing(client *mongo.Client, listingID string) error {
	collection := client.Database(dbName).Collection(listingsCollection)
	_, err := collection.DeleteOne(context.TODO(), bson.M{"_id": listingID})
	return err
}

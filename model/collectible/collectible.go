package collectible

import (
	"time"
)

// Rarity defines the rarity level of a collectible
type Rarity string

const (
	RarityCommon    Rarity = "Common"
	RarityUncommon  Rarity = "Uncommon"
	RarityRare      Rarity = "Rare"
	RarityEpic      Rarity = "Epic"
	RarityLegendary Rarity = "Legendary"
)

// Template represents a possible collectible item that can be minted
type Template struct {
	ID       string `bson:"_id,omitempty"`
	Name     string `bson:"name"`
	Rarity   Rarity `bson:"rarity"`
	Emoji    string `bson:"emoji"`     // Used for simple UI representation
	ImageURL string `bson:"image_url"` // Future support for GIFs/Images
}

// Item represents an actual minted instance of a collectible owned by a user
type Item struct {
	ID           string    `bson:"_id,omitempty"`
	TemplateID   string    `bson:"template_id"`
	SerialNumber int       `bson:"serial_number"`
	OwnerID      int       `bson:"owner_id"`
	MintedAt     time.Time `bson:"minted_at"`
}

// MarketListing represents an item currently listed for sale by a user
type MarketListing struct {
	ID       string    `bson:"_id,omitempty"`
	ItemID   string    `bson:"item_id"`
	SellerID int       `bson:"seller_id"`
	Price    int       `bson:"price"`
	ListedAt time.Time `bson:"listed_at"`
}

// CollectionScore assigns a base value to rarities for leaderboard purposes
var CollectionScore = map[Rarity]int{
	RarityCommon:    10,
	RarityUncommon:  25,
	RarityRare:      100,
	RarityEpic:      500,
	RarityLegendary: 2000,
}

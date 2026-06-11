package collectible

import (
	"errors"
	"math/rand"
	"time"

	model "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/model/collectible"
	repo "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository/collectible"
	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/repository"
	"go.mongodb.org/mongo-driver/mongo"
)

const PackPrice = 201

// OpenPack handles the logic for a user buying and opening a collectible pack
func OpenPack(client *mongo.Client, userID int, name string, chatID int64) (model.Item, model.Template, error) {
	// 1. Check user points
	currentPoints := repository.GetCurrentPoints(client, userID)
	if currentPoints < PackPrice {
		return model.Item{}, model.Template{}, errors.New("not enough points")
	}

	// 2. Deduct points
	repository.DeductWordlePoints(client, userID, name, chatID, PackPrice)

	// 3. Roll for rarity
	rarity := rollRarity()

	// 4. Fetch templates and filter by rolled rarity
	templates, err := repo.GetTemplates(client)
	if err != nil || len(templates) == 0 {
		// Fallback in case templates aren't loaded yet
		_ = repo.BootstrapTemplates(client)
		templates, _ = repo.GetTemplates(client)
	}

	var eligibleTemplates []model.Template
	for _, t := range templates {
		if t.Rarity == rarity {
			eligibleTemplates = append(eligibleTemplates, t)
		}
	}

	if len(eligibleTemplates) == 0 {
		return model.Item{}, model.Template{}, errors.New("no templates available for rolled rarity")
	}

	// 5. Select a random template from the eligible ones
	rand.Seed(time.Now().UnixNano())
	selectedTemplate := eligibleTemplates[rand.Intn(len(eligibleTemplates))]

	// 6. Get next serial number for this template
	serialNum, err := repo.GetNextSerialNumber(client, selectedTemplate.ID)
	if err != nil {
		repository.InsertWordleBonusDoc(userID, name, chatID, client, "WordleEn", PackPrice)
		return model.Item{}, model.Template{}, err
	}

	// 7. Mint the item
	newItem := model.Item{
		TemplateID:   selectedTemplate.ID,
		SerialNumber: serialNum,
		OwnerID:      userID,
		MintedAt:     time.Now(),
	}

	mintedItem, err := repo.MintItem(client, newItem)
	if err != nil {
		repository.InsertWordleBonusDoc(userID, name, chatID, client, "WordleEn", PackPrice)
		return model.Item{}, model.Template{}, err
	}

	return mintedItem, selectedTemplate, nil
}

// rollRarity simulates the gacha roll based on predefined probabilities
func rollRarity() model.Rarity {
	rand.Seed(time.Now().UnixNano())
	roll := rand.Intn(100) // 0 to 99

	switch {
	case roll < 60: // 60%
		return model.RarityCommon
	case roll < 85: // 25% (60 + 25 = 85)
		return model.RarityUncommon
	case roll < 95: // 10% (85 + 10 = 95)
		return model.RarityRare
	case roll < 99: // 4% (95 + 4 = 99)
		return model.RarityEpic
	default: // 1%
		return model.RarityLegendary
	}
}

// GetUserInventoryWithTemplates returns items alongside their templates for UI rendering
func GetUserInventoryWithTemplates(client *mongo.Client, userID int) ([]model.Item, map[string]model.Template, error) {
	items, err := repo.GetUserInventory(client, userID)
	if err != nil {
		return nil, nil, err
	}

	templates, err := repo.GetTemplates(client)
	if err != nil {
		return nil, nil, err
	}

	templateMap := make(map[string]model.Template)
	for _, t := range templates {
		templateMap[t.ID] = t
	}

	return items, templateMap, nil
}

// GetMarketplaceListingsWithDetails returns active listings alongside item and template info
func GetMarketplaceListingsWithDetails(client *mongo.Client) ([]model.MarketListing, map[string]model.Item, map[string]model.Template, error) {
	listings, err := repo.GetListings(client)
	if err != nil {
		return nil, nil, nil, err
	}

	templates, err := repo.GetTemplates(client)
	if err != nil {
		return nil, nil, nil, err
	}
	templateMap := make(map[string]model.Template)
	for _, t := range templates {
		templateMap[t.ID] = t
	}

	itemMap := make(map[string]model.Item)
	for _, listing := range listings {
		item, err := repo.GetItemByID(client, listing.ItemID)
		if err == nil {
			itemMap[item.ID] = item
		}
	}

	return listings, itemMap, templateMap, nil
}

// BuyItemFromMarketplace handles point deduction, seller reward, and ownership transfer
func BuyItemFromMarketplace(client *mongo.Client, listingID string, buyerID int, buyerName string, chatID int64) error {
	listing, err := repo.GetListingByID(client, listingID)
	if err != nil {
		return errors.New("listing not found")
	}

	if listing.SellerID == buyerID {
		return errors.New("you cannot buy your own listing")
	}

	// Check buyer points
	currentPoints := repository.GetCurrentPoints(client, buyerID)
	if currentPoints < listing.Price {
		return errors.New("not enough points")
	}

	// 1. Deduct points from buyer
	repository.DeductWordlePoints(client, buyerID, buyerName, chatID, listing.Price)

	// 2. Transfer item ownership & delete listing (simulate atomic action)
	err = repo.ProcessPurchase(client, listingID, listing.ItemID, buyerID)
	if err != nil {
		// If this fails, we should ideally refund the buyer.
		// For MVP, we refund immediately if transfer fails.
		repository.InsertWordleBonusDoc(buyerID, buyerName, chatID, client, "WordleEn", listing.Price)
		return err
	}

	// 3. Credit points to seller
	// Note: We don't have the seller's name or chatID easily accessible here, so we might insert a placeholder or look it up.
	// For now, we use a placeholder "Market Sale" name and a default chatID (0).
	// The DB point aggregation only strictly requires UserID.
	repository.InsertWordleBonusDoc(listing.SellerID, "Market Sale", 0, client, "WordleEn", listing.Price)

	return nil
}

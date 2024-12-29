package main

import (
	"log"
	"sync"

	"github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller"
	categorybot "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/categoryBot"
	instagrambot "github.com/MUSTAFA-A-KHAN/telegram-bot-anime/controller/instagramBot"
)

func main() {
	charades := "6898558980:AAFswe1zihkO2xAnKRQCAoLzt0gLW1uMJ88"
	categorycharades := "7563609270:AAFG12c-eWTKt8L2v_IN8pPF-HsmDDZGVgo"
	instagram := "7995903003:AAEcvtxq1Swak9W_uuMwQ-Jv-YXKOp_i-pw"
	// WaitGroup to wait for all goroutines to finish
	var wg sync.WaitGroup

	// Add two tasks to the WaitGroup
	wg.Add(3)

	// Start bots in separate goroutines
	go runWordBot(charades, &wg)
	go runCategoryBot(categorycharades, &wg)
	go runInstagramBot(instagram, &wg)

	// Wait for goroutines to complete
	wg.Wait()

	log.Println("All bots stopped running.")
}

// Function to start a bot with error handling
func runWordBot(botToken string, wg *sync.WaitGroup) {
	defer wg.Done() // Decrement the wait group counter when this goroutine completes
	err := controller.StartBot(botToken)
	if err != nil {
		log.Printf("Error starting bot with token %s: %v\n", botToken, err)
	}

}

// Function to start a bot with error handling
func runInstagramBot(botToken string, wg *sync.WaitGroup) {
	defer wg.Done() // Decrement the wait group counter when this goroutine completes
	err := instagrambot.StartBot(botToken)
	if err != nil {
		log.Printf("Error starting bot with token %s: %v\n", botToken, err)
	}

}

// // Function to start a bot with error handling
// func runAnimeBot(botToken string, wg *sync.WaitGroup) {
// 	defer wg.Done() // Decrement the wait group counter when this goroutine completes
// 	word, err := model.GetRandomCar()
// 	if err != nil {
// 		log.Printf("Error starting bot with token %s: %v\n", botToken, err)
// 	}
// 	fmt.Println("word", word)

// }

// Function to start a bot with error handling
func runCategoryBot(botToken string, wg *sync.WaitGroup) {
	defer wg.Done() // Decrement the wait group counter when this goroutine completes
	err := categorybot.StartBot(botToken)
	if err != nil {
		log.Printf("Error starting bot with token %s: %v\n", botToken, err)
	}

}

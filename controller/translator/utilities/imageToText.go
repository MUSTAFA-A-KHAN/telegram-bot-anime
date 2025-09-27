package utilities

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Struct to parse the response from the API
type TextData struct {
	Text        string `json:"text"`
	BoundingBox struct {
		X1 int `json:"x1"`
		Y1 int `json:"y1"`
		X2 int `json:"x2"`
		Y2 int `json:"y2"`
	} `json:"bounding_box"`
}

// Helper function to save the image
func SaveImage(bot *tgbotapi.BotAPI, chatID int64, photo tgbotapi.PhotoSize) (string, error) {
	// Get the file ID of the photo
	fileID := photo.FileID

	// Get file information from Telegram API
	file, err := bot.GetFile(tgbotapi.FileConfig{FileID: fileID})
	if err != nil {
		log.Printf("Failed to get file info: %v", err)
		return "", err
	}

	// Prepare file URL
	fileURL := fmt.Sprintf("https://api.telegram.org/file/bot%s/%s", bot.Token, file.FilePath)

	// Create a file to save the image
	outputPath := filepath.Join("images", file.FilePath[strings.LastIndex(file.FilePath, "/")+1:])

	// // Open a file to save the image
	// outputFile, err := os.Create(outputPath)
	// if err != nil {
	// 	log.Printf("Failed to create output file: %v", err)
	// 	return "", err
	// }
	// defer outputFile.Close()

	// Download the image using the URL and write to the output file
	err = DownloadFile(outputPath, fileURL)
	if err != nil {
		log.Printf("Failed to download file: %v", err)
		return "", err
	}

	log.Printf("Image saved as %s", outputPath)
	return outputPath, nil
}

func DownloadFile(filepath string, url string) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// // Function to download a file from a URL
// func downloadFile(url string, file *os.File) error {
// 	// Get the response from the URL
// 	resp, err := http.Get(url)
// 	if err != nil {
// 		return err
// 	}
// 	defer resp.Body.Close()

// 	// Write the response body (the image) to the output file
// 	_, err = io.Copy(file, resp.Body)
// 	return err
// }

func ImageToText(imagePath string, APINinjas string) string {
	fmt.Println(imagePath, "--------------------------")

	form := new(bytes.Buffer)
	writer := multipart.NewWriter(form)
	fw, err := writer.CreateFormFile("image", filepath.Base(imagePath))
	if err != nil {
		log.Fatal(err)
	}
	fd, err := os.Open(imagePath)
	if err != nil {
		log.Fatal(err)
	}
	defer fd.Close()
	_, err = io.Copy(fw, fd)
	if err != nil {
		log.Fatal(err)
	}

	writer.Close()

	// Send POST request to API Ninjas
	client := &http.Client{}
	req, err := http.NewRequest("POST", "https://api.api-ninjas.com/v1/imagetotext", form)
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("X-Api-Key", APINinjas)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	// Read and parse the response body
	bodyText, err := io.ReadAll(resp.Body)
	fmt.Println(bodyText)
	if err != nil {
		log.Fatal(err)
	}

	var response []TextData
	if err := json.Unmarshal(bodyText, &response); err != nil {
		log.Fatal(err)
	}

	// Print the extracted text
	fmt.Println("Extracted Text:")
	imgToTxt := ""
	for _, item := range response {
		fmt.Print(item.Text + " ")
		imgToTxt += item.Text + " "
	}
	return imgToTxt

}

package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

type Audio struct {
	Data        []byte
	ContentType string
	Filename    string
}

type TextTranslator struct{}

func NewTextTranslator() *TextTranslator {
	return &TextTranslator{}
}

func (t *TextTranslator) ReadItLoud(text string) Audio {
	// Using Google Text-to-Speech API
	url := fmt.Sprintf("https://translate.google.com/translate_tts?ie=UTF-8&tl=en&client=tw-ob&q=%s", text)

	resp, err := http.Get(url)
	if err != nil {
		return Audio{Data: []byte(fmt.Sprintf("TTS Error: %v", err)), ContentType: "text/plain", Filename: "error.txt"}
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return Audio{Data: []byte(fmt.Sprintf("TTS API Error: %d", resp.StatusCode)), ContentType: "text/plain", Filename: "error.txt"}
	}

	audioData, err := io.ReadAll(resp.Body)
	if err != nil {
		return Audio{Data: []byte(fmt.Sprintf("Error reading audio: %v", err)), ContentType: "text/plain", Filename: "error.txt"}
	}

	return Audio{
		Data:        audioData,
		ContentType: "audio/mpeg",
		Filename:    "speech.mp3",
	}
}

func (t *TextTranslator) WriteITDown(audioData []byte, contentType string) string {
	if AssemblyAIKey == "" {
		return "AssemblyAI API key not configured"
	}

	// Step 1: Upload the audio file
	uploadURL, err := t.uploadAudioToAssemblyAI(audioData)
	if err != nil {
		return fmt.Sprintf("Upload error: %v", err)
	}

	// Step 2: Submit transcription request
	transcriptID, err := t.submitTranscriptionRequest(uploadURL)
	if err != nil {
		return fmt.Sprintf("Transcription request error: %v", err)
	}

	// Step 3: Poll for transcription result
	transcript, err := t.pollTranscriptionResult(transcriptID)
	if err != nil {
		return fmt.Sprintf("Transcription polling error: %v", err)
	}

	return transcript
}

func (t *TextTranslator) uploadAudioToAssemblyAI(audioData []byte) (string, error) {
	url := "https://api.assemblyai.com/v2/upload"

	req, err := http.NewRequest("POST", url, bytes.NewReader(audioData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", AssemblyAIKey)
	req.Header.Set("Content-Type", "application/octet-stream")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("upload failed: %d - %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if uploadURL, ok := response["upload_url"].(string); ok {
		return uploadURL, nil
	}

	return "", fmt.Errorf("upload URL not found in response")
}

func (t *TextTranslator) submitTranscriptionRequest(audioURL string) (string, error) {
	url := "https://api.assemblyai.com/v2/transcript"

	payload := map[string]interface{}{
		"audio_url": audioURL,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", AssemblyAIKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("transcription request failed: %d - %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if transcriptID, ok := response["id"].(string); ok {
		return transcriptID, nil
	}

	return "", fmt.Errorf("transcript ID not found in response")
}

func (t *TextTranslator) pollTranscriptionResult(transcriptID string) (string, error) {
	url := fmt.Sprintf("https://api.assemblyai.com/v2/transcript/%s", transcriptID)

	for {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return "", err
		}

		req.Header.Set("Authorization", AssemblyAIKey)

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			return "", err
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			return "", err
		}

		if resp.StatusCode != 200 {
			return "", fmt.Errorf("polling failed: %d - %s", resp.StatusCode, string(body))
		}

		var response map[string]interface{}
		if err := json.Unmarshal(body, &response); err != nil {
			return "", err
		}

		if status, ok := response["status"].(string); ok {
			if status == "completed" {
				if text, ok := response["text"].(string); ok {
					return text, nil
				}
				return "", fmt.Errorf("text not found in completed response")
			} else if status == "error" {
				return "", fmt.Errorf("transcription error: %v", response["error"])
			}
		}

		// Wait before polling again
		// In production, use time.Sleep or a proper polling mechanism
	}
}

func (t *TextTranslator) TranslateToEnglish(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Translate the following text to English and just reply with the translated text: %s", text)},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OpenAIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Sprintf("API Error: %d - %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Sprintf("Parse error: %v", err)
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	return "Translation failed"
}

func (t *TextTranslator) TranslateToArabic(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Translate the following text to Arabic and just reply with the translated text: %s", text)},
		},
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", OpenAIKey))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Sprintf("Error: %v", err)
	}

	if resp.StatusCode != 200 {
		return fmt.Sprintf("API Error: %d - %s", resp.StatusCode, string(body))
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Sprintf("Parse error: %v", err)
	}

	if choices, ok := response["choices"].([]interface{}); ok && len(choices) > 0 {
		if choice, ok := choices[0].(map[string]interface{}); ok {
			if message, ok := choice["message"].(map[string]interface{}); ok {
				if content, ok := message["content"].(string); ok {
					return content
				}
			}
		}
	}

	return "Translation failed"
}

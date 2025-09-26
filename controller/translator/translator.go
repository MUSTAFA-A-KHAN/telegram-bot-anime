package translator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
)

type Audio struct {
	Data        []byte
	ContentType string
	Filename    string
}
type Response struct {
	AudioFile string `json:"audioFile"`
}

// Implement the io.Reader interface for Audio
func (a *Audio) Read(p []byte) (n int, err error) {
	// Copy the audio data into the provided slice and return the number of bytes read
	n = copy(p, a.Data)
	if n < len(a.Data) {
		return n, io.ErrShortBuffer
	}
	return n, nil
}

type TextTranslator struct{}

func NewTextTranslator() *TextTranslator {
	return &TextTranslator{}
}

func (t *TextTranslator) ReadItLoudUK(text string) string {
	client := &http.Client{}

	// Build JSON properly
	reqBody := map[string]interface{}{
		"text":              text,
		"voice_id":          "en-AU-leyton",
		"multiNativeLocale": "en-UK",
		"style":             "Angry",
		"pronunciationDictionary": map[string]map[string]string{
			"2010": {
				"pronunciation": "two thousand and ten",
				"type":          "SAY_AS",
			},
			"live": {
				"pronunciation": "laɪv",
				"type":          "IPA",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://api.murf.ai/v1/speech/generate", bytes.NewReader(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", "ap2_23dcac5c-ad9f-4435-877e-4706abf4a9e3")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result Response
	if err := json.Unmarshal(bodyText, &result); err != nil {
		log.Fatal(err)
	}
	if err := downloadFile("outputUK.mp3", result.AudioFile); err != nil {
		log.Fatal(err)
	}

	return result.AudioFile
}

func (t *TextTranslator) ReadItLoudUKFemale(text string) string {
	client := &http.Client{}

	// Build JSON properly
	reqBody := map[string]interface{}{
		"text":              text,
		"voice_id":          "en-US-samantha",
		"style":             "Luxury",
		"multiNativeLocale": "en-UK",
		"pronunciationDictionary": map[string]map[string]string{
			"2010": {
				"pronunciation": "two thousand and ten",
				"type":          "SAY_AS",
			},
			"live": {
				"pronunciation": "laɪv",
				"type":          "IPA",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://api.murf.ai/v1/speech/generate", bytes.NewReader(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", "ap2_23dcac5c-ad9f-4435-877e-4706abf4a9e3")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result Response
	if err := json.Unmarshal(bodyText, &result); err != nil {
		log.Fatal(err)
	}
	if err := downloadFile("outputUKFemale.mp3", result.AudioFile); err != nil {
		log.Fatal(err)
	}

	return result.AudioFile
}

func (t *TextTranslator) ReadItLoud(text string) string {
	client := &http.Client{}

	// Build JSON properly
	reqBody := map[string]interface{}{
		"text":    text,
		"voiceId": "en-US-charles",
		"pronunciationDictionary": map[string]map[string]string{
			"2010": {
				"pronunciation": "two thousand and ten",
				"type":          "SAY_AS",
			},
			"live": {
				"pronunciation": "laɪv",
				"type":          "IPA",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://api.murf.ai/v1/speech/generate", bytes.NewReader(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", "ap2_23dcac5c-ad9f-4435-877e-4706abf4a9e3")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result Response
	if err := json.Unmarshal(bodyText, &result); err != nil {
		log.Fatal(err)
	}
	if err := downloadFile("output.mp3", result.AudioFile); err != nil {
		log.Fatal(err)
	}

	return result.AudioFile
}
func (t *TextTranslator) ReadItLoudFemale(text string) string {
	client := &http.Client{}

	// Build JSON properly
	reqBody := map[string]interface{}{
		"text":    text,
		"voiceId": "en-US-natalie",
		"pronunciationDictionary": map[string]map[string]string{
			"2010": {
				"pronunciation": "two thousand and ten",
				"type":          "SAY_AS",
			},
			"live": {
				"pronunciation": "laɪv",
				"type":          "IPA",
			},
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequest("POST", "https://api.murf.ai/v1/speech/generate", bytes.NewReader(jsonData))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("api-key", "ap2_23dcac5c-ad9f-4435-877e-4706abf4a9e3")

	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()

	bodyText, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	var result Response
	if err := json.Unmarshal(bodyText, &result); err != nil {
		log.Fatal(err)
	}
	if err := downloadFile("outputFemale.mp3", result.AudioFile); err != nil {
		log.Fatal(err)
	}

	return result.AudioFile
}

func downloadFile(filepath string, url string) error {
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

func (t *TextTranslator) TranslateToRussian(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Translate the following text to Russian and just reply with the translated text: %s", text)},
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

func (t *TextTranslator) TranslateToFrench(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Translate the following text to French and just reply with the translated text: %s", text)},
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

func (t *TextTranslator) GetSynonyms(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Give me synonyms for the word and please just give me synonyms nothing else it has to be atleast 30 synonyms of the word: %s", text)},
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

func (t *TextTranslator) GetAntonyms(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Give me synonyms for the word and please just give me antonyms nothing else it has to be atleast 30 antonyms of the word: %s", text)},
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

func (t *TextTranslator) GetDefinition(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Provide a clear and concise definition of the word/phrase/idiom %s. Include its meaning, common usage, and any variations or related terms", text)},
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

func (t *TextTranslator) GetAbbreviation(text string) string {
	if OpenAIKey == "" {
		return "OpenAI API key not configured"
	}

	url := "https://glama.ai/api/gateway/openai/v1/chat/completions"

	payload := map[string]interface{}{
		"model": "openai/gpt-4o",
		"messages": []map[string]string{
			{"role": "user", "content": fmt.Sprintf("Convert the word/phrase %s into its commonly used abbreviation or acronym. If multiple abbreviations exist, provide the most widely accepted ones.", text)},
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

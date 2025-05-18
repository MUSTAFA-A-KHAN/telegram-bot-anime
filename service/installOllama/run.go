package service

import (
	"bytes"
	"os/exec"
)

func RunOllama(prompt string) (string, error) {
	// Prepare the command
	cmd := exec.Command("ollama", "run", "tinyllama")

	// Create a buffer to hold the input (the prompt)
	var stdin bytes.Buffer
	stdin.WriteString(prompt)

	// Connect stdin of the command to the buffer
	cmd.Stdin = &stdin

	// Capture the output
	var out bytes.Buffer
	cmd.Stdout = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

	// Return the output as string
	return out.String(), nil
}

// func main() {
// 	response, err := runOllama("Hello, how are you?")
// 	if err != nil {
// 		fmt.Println("Error:", err)
// 		return
// 	}
// 	fmt.Println("Response from Ollama:")
// 	fmt.Println(response)
// }

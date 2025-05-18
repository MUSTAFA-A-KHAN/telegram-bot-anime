package service

import (
	"bytes"
	"os/exec"
	"strings"
)

func RunOllama(prompt string) (<-chan string, <-chan error) {
	lineChannel := make(chan string)
	errChannel := make(chan error, 1)

	cmd := exec.Command("ollama", "run", "super-mario-llama")

	// Set up input
	cmd.Stdin = bytes.NewBufferString(prompt)

	// Capture the output
	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		errChannel <- err
		close(lineChannel)
		close(errChannel)
		return lineChannel, errChannel
	}

	// Start the command
	go func() {
		defer close(lineChannel)
		defer close(errChannel)

		if err := cmd.Start(); err != nil {
			errChannel <- err
			return
		}

		buf := make([]byte, 1024)
		for {
			n, err := stdoutPipe.Read(buf)
			if err != nil {
				errChannel <- err
				break
			}
			// Split the buffer into lines and send them
			for _, line := range strings.Split(string(buf[:n]), "\n") {
				// Filter out empty lines if necessary
				if strings.TrimSpace(line) != "" {
					lineChannel <- line
				}
			}
		}
		cmd.Wait()
	}()

	return lineChannel, errChannel
}

func BuildOllamaModel() (string, error) {
	// Command to build the model from Modelfile
	cmd := exec.Command("ollama", "create", "super-mario-llama", "-f", "Modelfile")

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	// Run the command
	err := cmd.Run()
	if err != nil {
		return "", err
	}

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

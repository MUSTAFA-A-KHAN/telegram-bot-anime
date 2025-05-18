package service

import (
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"time"
)

func Install() ([]string, error) {
	var logs []string

	logs = append(logs, "🌟 Starting Ollama full setup...")

	// Step 1: Download the Ollama install script
	resp, err := http.Get("https://ollama.com/install.sh")
	if err != nil {
		return logs, fmt.Errorf("❌ Could not download install script: %v", err)
	}
	defer resp.Body.Close()

	scriptPath := "/tmp/install_ollama.sh"
	out, err := os.Create(scriptPath)
	if err != nil {
		return logs, fmt.Errorf("❌ Cannot create script file: %v", err)
	}
	io.Copy(out, resp.Body)
	out.Close()

	// Step 2: Make the script executable
	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		return logs, fmt.Errorf("❌ Cannot make script executable: %v", err)
	}

	// Step 3: Run the installer
	logs = append(logs, "🔧 Installing Ollama...")
	cmd := exec.Command("sh", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		return logs, fmt.Errorf("❌ Ollama installation failed: %v", err)
	}

	// Step 4: Start Ollama in the background
	logs = append(logs, "🚀 Launching Ollama server...")
	serve := exec.Command("ollama", "serve")
	serve.Stdout = os.Stdout
	serve.Stderr = os.Stderr
	err = serve.Start()
	if err != nil {
		return logs, fmt.Errorf("❌ Ollama server failed to start: %v", err)
	}

	// Step 5: Wait until Ollama is ready
	logs = append(logs, "⏳ Waiting for Ollama to be ready")
	for {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:11434", 2*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		time.Sleep(1 * time.Second)
	}
	logs = append(logs, "✅ Ollama is ready!")

	// Step 6: Pull the llama3 model
	logs = append(logs, "📥 Pulling model (llama3)...")
	pull := exec.Command("ollama", "pull", "llama3")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	err = pull.Run()
	if err != nil {
		return logs, fmt.Errorf("❌ Model pull failed: %v", err)
	}

	logs = append(logs, "🎉 All done! Ollama is installed, running, and ready to answer your questions.")
	return logs, nil
}

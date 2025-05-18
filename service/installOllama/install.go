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

func Install() {
	fmt.Println("🌟 Starting Ollama full setup...")

	// Step2: Download the Ollama install script
	resp, err := http.Get("https://ollama.com/install.sh")
	if err != nil {
		panic("❌ Could not download install script: " + err.Error())
	}
	defer resp.Body.Close()

	scriptPath := "/tmp/install_ollama.sh"
	out, err := os.Create(scriptPath)
	if err != nil {
		panic("❌ Cannot create script file: " + err.Error())
	}
	io.Copy(out, resp.Body)
	out.Close()

	// Step 2: Make the script executable
	err = os.Chmod(scriptPath, 0755)
	if err != nil {
		panic("❌ Cannot make script executable: " + err.Error())
	}

	// Step 3: Run the installer
	fmt.Println("🔧 Installing Ollama...")
	cmd := exec.Command("sh", scriptPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		panic("❌ Ollama installation failed: " + err.Error())
	}

	// Step 4: Start Ollama in the background
	fmt.Println("🚀 Launching Ollama server...")
	serve := exec.Command("ollama", "serve")
	serve.Stdout = os.Stdout
	serve.Stderr = os.Stderr
	err = serve.Start()
	if err != nil {
		panic("❌ Ollama server failed to start: " + err.Error())
	}

	// Step 5: Wait until Ollama is ready (port 11434 open)
	fmt.Print("⏳ Waiting for Ollama to be ready")
	for {
		conn, err := net.DialTimeout("tcp", "127.0.0.1:11434", 2*time.Second)
		if err == nil {
			conn.Close()
			break
		}
		fmt.Print(".")
		time.Sleep(1 * time.Second)
	}
	fmt.Println("\n✅ Ollama is ready!")

	// Step 6: Pull the llama3 model
	fmt.Println("📥 Pulling model (llama3)...")
	pull := exec.Command("ollama", "pull", "llama3")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	err = pull.Run()
	if err != nil {
		panic("❌ Model pull failed: " + err.Error())
	}

	fmt.Println("🎉 All done! Ollama is installed, running, and ready to answer your questions.")
}

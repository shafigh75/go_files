// example of a cicd pipeline in golang: 

package main

import (
	"bytes"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func main() {
	logger := log.New(os.Stdout, "🚦 ", log.Ltime)

	logger.Println("Starting CI/CD Pipeline")
	defer logger.Println("Pipeline terminated")

	steps := []struct {
		name string
		fn   func() error
	}{
		{"Build", build},
		{"Test", test},
		{"Deploy", deploy},
	}

	for _, step := range steps {
		logger.Printf("Stage: %s\n", step.name)
		if err := step.fn(); err != nil {
			logger.Fatalf("❌ %s failed: %v", step.name, err)
		}
	}
}

func build() error {
	return runCommand("go build -o app .", "🔨")
}

func test() error {
	return runCommand("go test ./...", "🧪")
}

func deploy() error {
	return runCommand("docker build -t myapp:latest .", "🚀")
}

func runCommand(command string, emoji string) error {
	cmd := exec.Command("sh", "-c", command)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out

	fmt.Printf("%s Executing: %s\n", emoji, command)
	if err := cmd.Run(); err != nil {
		fmt.Printf("Command output:\n%s\n", out.String())
		return err
	}
	return nil
}

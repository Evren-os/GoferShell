package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"
	"github.com/chzyer/readline"
)

// ANSI color codes for terminal output
const (
	blue   = "\033[34m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
)

func main() {
	// Verify ytmax executable exists in PATH
	if _, err := exec.LookPath("ytmax"); err != nil {
		fmt.Println(red + "Error: ytmax executable not found in PATH" + reset)
		os.Exit(1)
	}

	// Initialize readline with a colored prompt
	rl, err := readline.New(blue + "Enter video URLs (separated by commas). Press [ENTER] when done: " + reset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing readline:", err)
		os.Exit(1)
	}
	defer rl.Close()

	// Read the input line
	urls, err := rl.Readline()
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	// Trim whitespace and check if input is empty
	urls = strings.TrimSpace(urls)
	if urls == "" {
		fmt.Println(yellow + "No URLs entered. Exiting." + reset)
		os.Exit(0)
	}

	// Split URLs by comma and process each
	urlList := strings.Split(urls, ",")
	var failedURLs []string
	for _, url := range urlList {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		// Indicate download start
		fmt.Println("\n" + yellow + "Downloading: " + url + reset)

		// Execute ytmax for the URL
		cmd := exec.Command("ytmax", url)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			// Handle failure
			fmt.Println(red + "Failed to download: " + url + reset)
			failedURLs = append(failedURLs, url)
		}
	}

	// Report any failed URLs
	if len(failedURLs) > 0 {
		fmt.Println("\n" + red + "Failed URLs:" + reset)
		for _, url := range failedURLs {
			fmt.Println(url)
		}
	}
}

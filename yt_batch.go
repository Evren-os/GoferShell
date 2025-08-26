package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/chzyer/readline"
)

// ANSI color codes for terminal output.
const (
	colorBlue   = "\033[34m"
	colorYellow = "\033[33m"
	colorRed    = "\033[31m"
	colorReset  = "\033[0m"
)

// fatalf prints a formatted error message to stderr and exits with status 1.
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// getURLsFromInput prompts the user for a comma-separated list of URLs using readline.
func getURLsFromInput() []string {
	rl, err := readline.New(colorBlue + "Enter video URLs (comma-separated), then press [ENTER]: " + colorReset)
	if err != nil {
		fatalf("failed to initialize readline: %v", err)
	}
	defer rl.Close()

	line, err := rl.Readline()
	if err != nil {
		if err == readline.ErrInterrupt || err.Error() == "EOF" {
			fmt.Println(colorYellow + "\nInput cancelled. Exiting." + colorReset)
			os.Exit(0)
		}
		fatalf("failed to read input: %v", err)
	}

	trimmedLine := strings.TrimSpace(line)
	if trimmedLine == "" {
		fmt.Println(colorYellow + "No URLs entered. Exiting." + colorReset)
		os.Exit(0)
	}

	var urls []string
	for _, url := range strings.Split(trimmedLine, ",") {
		if cleanURL := strings.TrimSpace(url); cleanURL != "" {
			urls = append(urls, cleanURL)
		}
	}
	return urls
}

// downloadURL executes the ytmax command for a single URL in a goroutine.
func downloadURL(url string, baseArgs []string, wg *sync.WaitGroup, sem chan struct{}, failedURLsChan chan<- string) {
	defer wg.Done()
	defer func() { <-sem }() // Release semaphore slot.

	fmt.Printf("%sStarting download:%s %s\n", colorYellow, colorReset, url)

	fullArgs := append(baseArgs, url)
	cmd := exec.Command("ytmax", fullArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("%sFailed to download:%s %s\n", colorRed, colorReset, url)
		failedURLsChan <- url
	}
}

func main() {
	// Define command-line flags to mirror ytmax and control concurrency.
	var (
		downloadDir string
		codecPref   string
		cookiesFrom string
		socm        bool
		parallel    int
	)

	flag.StringVar(&downloadDir, "d", "", "Download destination directory.")
	flag.StringVar(&codecPref, "codec", "av1", "Preferred video codec (av1 or vp9). Ignored if -socm is used.")
	flag.StringVar(&cookiesFrom, "cookies-from", "", "Load cookies from the specified browser (e.g., firefox, chrome).")
	flag.BoolVar(&socm, "socm", false, "Optimize for social media compatibility (MP4, H.264/AAC).")
	flag.IntVar(&parallel, "p", 4, "Number of parallel downloads.")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: yt_batch [options]\n\n")
		fmt.Fprintf(out, "A tool for batch downloading videos concurrently using ytmax.\n")
		fmt.Fprintf(out, "It will prompt for a comma-separated list of URLs.\n\n")
		fmt.Fprintf(out, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if parallel < 1 {
		fatalf("number of parallel downloads (-p) must be at least 1")
	}

	if _, err := exec.LookPath("ytmax"); err != nil {
		fatalf("ytmax executable not found in PATH")
	}

	urls := getURLsFromInput()

	// Build the base command arguments to pass to each ytmax instance.
	var baseCmdArgs []string
	if downloadDir != "" {
		baseCmdArgs = append(baseCmdArgs, "-d", downloadDir)
	}
	if cookiesFrom != "" {
		baseCmdArgs = append(baseCmdArgs, "--cookies-from", cookiesFrom)
	}
	if socm {
		baseCmdArgs = append(baseCmdArgs, "-socm")
	} else {
		baseCmdArgs = append(baseCmdArgs, "-codec", codecPref)
	}

	// Setup for concurrent processing.
	var wg sync.WaitGroup
	sem := make(chan struct{}, parallel)
	failedURLsChan := make(chan string, len(urls))

	for _, url := range urls {
		wg.Add(1)
		sem <- struct{}{}
		go downloadURL(url, baseCmdArgs, &wg, sem, failedURLsChan)
	}

	wg.Wait()
	close(failedURLsChan)

	var failedURLs []string
	for url := range failedURLsChan {
		failedURLs = append(failedURLs, url)
	}

	if len(failedURLs) > 0 {
		fmt.Printf("\n%s--- Summary ---%s\n", colorRed, colorReset)
		fmt.Printf("%d/%d downloads failed.\n", len(failedURLs), len(urls))
		fmt.Println("Failed URLs:")
		for _, url := range failedURLs {
			fmt.Printf("  - %s\n", url)
		}
	} else {
		fmt.Printf("\n%s--- Summary ---%s\n", colorBlue, colorReset)
		fmt.Printf("All %d downloads completed successfully.\n", len(urls))
	}
}
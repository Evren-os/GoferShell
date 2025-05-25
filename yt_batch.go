package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/chzyer/readline"
)

const (
	blue   = "\033[34m"
	yellow = "\033[33m"
	red    = "\033[31m"
	reset  = "\033[0m"
)

// Usage message
func printHelp() {
	fmt.Printf("%syt_batch%s: Effortless batch video downloading via ytmax.\n\n", yellow, reset)
	fmt.Printf("%sUSAGE%s:\n", yellow, reset)
	fmt.Printf("  %syt_batch%s [options]\n\n", blue, reset)
	fmt.Printf("%sDESCRIPTION%s:\n", yellow, reset)
	fmt.Printf("  yt_batch simplifies downloading multiple videos. It prompts for a\n")
	fmt.Printf("  comma-separated list of URLs and processes each using 'ytmax'.\n\n")
	fmt.Printf("%sOPTIONS%s:\n", yellow, reset)
	fmt.Printf("  %s-d <directory>%s    Set the download destination directory.\n", blue, reset)
	fmt.Printf("                      If not specified, videos are saved to the\n")
	fmt.Printf("                      current working directory.\n")
	fmt.Printf("  %s--help%s             Display this help message and exit.\n\n", blue, reset)
	fmt.Printf("%sEXAMPLES%s:\n", yellow, reset)
	fmt.Printf("  %syt_batch%s\n", blue, reset)
	fmt.Printf("    (Prompts for URLs, downloads to current directory)\n\n")
	fmt.Printf("  %syt_batch -d /mnt/media/new_videos%s\n", blue, reset)
	fmt.Printf("    (Prompts for URLs, downloads to /mnt/media/new_videos)\n\n")
	fmt.Printf("%sREQUIREMENTS%s:\n", yellow, reset)
	fmt.Printf("  - 'ytmax' must be installed and accessible in your PATH.\n")
}

func main() {
	var downloadDir string
	var showHelp bool

	flag.StringVar(&downloadDir, "d", "", "Download destination directory.")
	flag.BoolVar(&showHelp, "help", false, "Show help message.")

	flag.Usage = printHelp

	flag.Parse()

	if showHelp {
		printHelp()
		os.Exit(0)
	}

	// Verify ytmax executable exists in PATH.
	if _, err := exec.LookPath("ytmax"); err != nil {
		fmt.Println(red + "Error: ytmax executable not found in PATH" + reset)
		os.Exit(1)
	}

	rl, err := readline.New(blue + "Enter video URLs (separated by commas). Press [ENTER] when done: " + reset)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error initializing readline:", err)
		os.Exit(1)
	}
	defer rl.Close()

	urls, err := rl.Readline()
	if err != nil {
		if err.Error() == "Interrupt" || err.Error() == "EOF" {
			fmt.Println(yellow + "\nInput cancelled. Exiting." + reset)
			os.Exit(0)
		}
		fmt.Fprintln(os.Stderr, "Error reading input:", err)
		os.Exit(1)
	}

	urls = strings.TrimSpace(urls)
	if urls == "" {
		fmt.Println(yellow + "No URLs entered. Exiting." + reset)
		os.Exit(0)
	}

	urlList := strings.Split(urls, ",")
	var failedURLs []string
	for _, url := range urlList {
		url = strings.TrimSpace(url)
		if url == "" {
			continue
		}

		fmt.Println("\n" + yellow + "Downloading: " + url + reset)

		var cmdArgs []string
		if downloadDir != "" {
			cmdArgs = append(cmdArgs, "-d", downloadDir)
		}
		cmdArgs = append(cmdArgs, url)

		cmd := exec.Command("ytmax", cmdArgs...)
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
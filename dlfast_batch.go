package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

func main() {
	// Verify dlfast executable exists in PATH
	if _, err := exec.LookPath("dlfast"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'dlfast' executable not found. Ensure it is in your PATH.")
		os.Exit(1)
	}

	// Parse command-line flags
	var targetDir string
	flag.StringVar(&targetDir, "d", "", "target directory")
	flag.Parse()

	// Get URLs from remaining arguments
	urls := flag.Args()
	if len(urls) == 0 {
		fmt.Println("Usage: dlfast_batch [-d target_directory] url1 [url2 ...]")
		fmt.Println("Example: dlfast_batch -d /path/to/dir \"http://example.com/file1.zip\" \"http://example.com/file2.zip\"")
		os.Exit(1)
	}

	// Handle optional target directory
	if targetDir != "" {
		absDir, err := filepath.Abs(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving target directory: %v\n", err)
			os.Exit(1)
		}
		if err := os.MkdirAll(absDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating target directory: %v\n", err)
			os.Exit(1)
		}
		info, err := os.Stat(absDir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: '%s' is not a directory\n", absDir)
			os.Exit(1)
		}
		targetDir = absDir
	}

	// Process each URL
	var success, failure []string
	for _, url := range urls {
		fmt.Println("→ Downloading:", url)
		args := []string{url}
		if targetDir != "" {
			args = append(args, targetDir)
		}
		cmd := exec.Command("dlfast", args...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				failure = append(failure, fmt.Sprintf("%s (exit %d)", url, exitErr.ExitCode()))
			} else {
				failure = append(failure, fmt.Sprintf("%s (error: %v)", url, err))
			}
		} else {
			success = append(success, url)
		}
	}

	// Print summary
	fmt.Println("\n===== Batch Download Summary =====")
	fmt.Printf("Total URLs: %d\n", len(urls))
	fmt.Printf("Succeeded:  %d\n", len(success))
	for _, u := range success {
		fmt.Println("  ✓", u)
	}
	if len(failure) > 0 {
		fmt.Printf("Failed:     %d\n", len(failure))
		for _, u := range failure {
			fmt.Println("  ✗", u)
		}
		os.Exit(1)
	} else {
		fmt.Println("All downloads completed successfully!")
		os.Exit(0)
	}
}

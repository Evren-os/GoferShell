package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func main() {
	// Check if URL is provided
	if len(os.Args) < 2 {
		fmt.Println("Usage: dlfast <URL> [target_directory_or_filepath]")
		fmt.Println("Example (to CWD): dlfast http://example.com/file.zip")
		fmt.Println("Example (to dir): dlfast http://example.com/file.zip /mnt/data/")
		fmt.Println("Example (to file): dlfast http://example.com/file.zip ~/Downloads/archive.zip")
		os.Exit(1)
	}

	url := os.Args[1]
	var destination string
	if len(os.Args) > 2 {
		destination = os.Args[2]
	}

	var targetDir string
	var aria2OutputOpts []string

	// Handle destination logic
	if destination == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error getting current working directory: %v\n", err)
			os.Exit(1)
		}
		targetDir = cwd
		aria2OutputOpts = []string{"--dir", targetDir}
		fmt.Println("No destination specified. Downloading to current directory:", targetDir)
		fmt.Println("Filename will be inferred from URL.")
	} else {
		// Resolve absolute path (shell expands ~, filepath.Abs handles relative paths)
		absDest, err := filepath.Abs(destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving destination path: %v\n", err)
			os.Exit(1)
		}

		if strings.HasSuffix(destination, "/") {
			// Treat as directory if ends with "/"
			targetDir = absDest
			aria2OutputOpts = []string{"--dir", targetDir}
			fmt.Println("Outputting to directory:", targetDir)
			fmt.Println("Filename will be inferred from URL.")
			fmt.Println("For best resume reliability with new links for this specific file, consider specifying the full output file path next time if the link changes.")
		} else {
			// Check if destination exists and is a directory
			info, err := os.Stat(absDest)
			if err == nil && info.IsDir() {
				targetDir = absDest
				aria2OutputOpts = []string{"--dir", targetDir}
				fmt.Println("Outputting to directory:", targetDir)
				fmt.Println("Filename will be inferred from URL.")
				fmt.Println("For best resume reliability with new links for this specific file, consider specifying the full output file path next time if the link changes.")
			} else {
				// Treat as file path
				targetDir = filepath.Dir(absDest)
				filename := filepath.Base(absDest)
				aria2OutputOpts = []string{"--dir", targetDir, "--out", filename}
				fmt.Println("Outputting to file:", absDest)
			}
		}
	}

	// Ensure target directory exists
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create directory '%s': %v\n", targetDir, err)
		os.Exit(1)
	}

	// Verify directory is writable
	tmpFile, err := os.CreateTemp(targetDir, ".dlfast-check-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Directory '%s' is not writable: %v\n", targetDir, err)
		os.Exit(1)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	// Common aria2c options
	aria2CommonOpts := []string{
		"--continue=true",
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
		"--file-allocation=falloc",
		"--max-tries=0",
		"--retry-wait=5",
		"--timeout=60",
		"--max-file-not-found=3",
		"--summary-interval=3",
		"--console-log-level=warn",
		"--auto-file-renaming=false",
		"--conditional-get=true",
		"--check-integrity=true",
		"--disk-cache=64M",
		"--allow-overwrite=true",
		"--async-dns=true",
		"--http-accept-gzip=true",
		"--remote-time=true",
	}

	// Construct aria2c arguments
	args := append(aria2CommonOpts, aria2OutputOpts...)
	args = append(args, url)

	// Execute aria2c
	fmt.Println("Starting download with aria2c...")
	cmd := exec.Command("aria2c", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err = cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "aria2c exited with status %d for %s.\n", exitErr.ExitCode(), url)
			os.Exit(exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stderr, "Error running aria2c: %v\n", err)
			os.Exit(1)
		}
	}
	fmt.Println("Download completed successfully:", url)
}

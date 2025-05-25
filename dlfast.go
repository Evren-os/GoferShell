package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
)

func main() {
	var destination string
	flag.StringVar(&destination, "d", "", "Target directory or full filepath for download. If not provided, downloads to current directory.")

	// Setup custom usage message
	flag.Usage = func() {
		cmdName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "dlfast: A basic CLI tool to download files using aria2.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [-d target_directory_or_filepath] <URL>\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s http://example.com/file.zip                     (Download to current directory)\n", cmdName)
		fmt.Fprintf(os.Stderr, "  %s -d /mnt/data/ http://example.com/file.zip         (Download to /mnt/data/, filename from URL)\n", cmdName)
		fmt.Fprintf(os.Stderr, "  %s -d ~/Downloads/archive.zip http://example.com/file.zip (Download to ~/Downloads/archive.zip)\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	url := flag.Arg(0)

	var targetDir string
	var aria2OutputOpts []string

	// Determine target directory and aria2c output options based on the -d flag
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
		absDest, err := filepath.Abs(destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving destination path: %v\n", err)
			os.Exit(1)
		}

		if strings.HasSuffix(destination, "/") {
			targetDir = absDest
			aria2OutputOpts = []string{"--dir", targetDir}
			fmt.Println("Outputting to directory:", targetDir)
			fmt.Println("Filename will be inferred from URL.")
			fmt.Println("For best resume reliability with new links for this specific file, consider specifying the full output file path next time if the link changes.")
		} else {
			info, err := os.Stat(absDest)
			if err == nil && info.IsDir() {
				targetDir = absDest
				aria2OutputOpts = []string{"--dir", targetDir}
				fmt.Println("Outputting to directory:", targetDir)
				fmt.Println("Filename will be inferred from URL.")
				fmt.Println("For best resume reliability with new links for this specific file, consider specifying the full output file path next time if the link changes.")
			} else {
				targetDir = filepath.Dir(absDest)
				filename := filepath.Base(absDest)
				aria2OutputOpts = []string{"--dir", targetDir, "--out", filename}
				fmt.Println("Outputting to file:", absDest)
			}
		}
	}

	// Ensure target directory exists and is writable
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create directory '%s': %v\n", targetDir, err)
		os.Exit(1)
	}

	tmpFile, err := os.CreateTemp(targetDir, ".dlfast-check-*")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Directory '%s' is not writable: %v\n", targetDir, err)
		os.Exit(1)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	// Common aria2c arguments for optimized downloads
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

	args := append(aria2CommonOpts, aria2OutputOpts...)
	args = append(args, url)

	ctx, cancel := context.WithCancel(context.Background())

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Goroutine to listen for signals and cancel the context
	go func() {
		select {
		case sig := <-sigChan:
			fmt.Fprintf(os.Stderr, "\nSignal (%s) received, attempting to cancel download...\n", sig)
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	fmt.Println("Starting download with aria2c...")
	cmd := exec.CommandContext(ctx, "aria2c", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	// Clean up signal handling and ensure context is cancelled
	signal.Stop(sigChan)
	close(sigChan)
	cancel()

	// Handle command execution result
	if err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Fprintf(os.Stderr, "Download for %s was cancelled by user.\n", url)
			os.Exit(130)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "aria2c exited with status %d for %s.\n", exitErr.ExitCode(), url)
			os.Exit(exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stderr, "Error running aria2c: %v\n", err)
			os.Exit(1)
		}
	} else {
		fmt.Println("Download completed successfully:", url)
	}
}
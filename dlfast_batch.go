package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
)

func main() {
	// Verify dlfast executable exists in PATH
	if _, err := exec.LookPath("dlfast"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'dlfast' executable not found. Ensure it is in your PATH and is the correct version with signal handling.")
		os.Exit(1)
	}

	var targetDir string
	flag.StringVar(&targetDir, "d", "", "Target directory for all downloads. If not provided, 'dlfast' will use its default (usually current directory).")

	// Usage message
	flag.Usage = func() {
		cmdName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: A tool to download multiple files in batch using 'dlfast'.\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Usage: %s [-d target_directory] <URL1> [URL2 ...]\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  URL1 [URL2 ...]    One or more URLs to download.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -d /path/to/downloads \"http://example.com/file1.zip\" \"http://example.com/file2.tar.gz\"\n", cmdName)
		fmt.Fprintf(os.Stderr, "\nNote: This tool relies on 'dlfast' being correctly installed and handling signals (like Ctrl+C) gracefully.\n")
	}

	flag.Parse()

	urls := flag.Args()
	if len(urls) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var absTargetDir string
	if targetDir != "" {
		var err error
		absTargetDir, err = filepath.Abs(targetDir)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error resolving target directory '%s': %v\n", targetDir, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(absTargetDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating target directory '%s': %v\n", absTargetDir, err)
			os.Exit(1)
		}
		info, err := os.Stat(absTargetDir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: Target path '%s' is not a valid directory.\n", absTargetDir)
			os.Exit(1)
		}
	}

	// Context and signal handling
	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Fprintf(os.Stderr, "\nSignal (%s) received in dlfast_batch, attempting to stop batch processing...\n", sig)
			mainCancel()
		case <-mainCtx.Done():
			return
		}
	}()

	var success, failure []string
	for i, url := range urls {
		if mainCtx.Err() != nil {
			fmt.Fprintf(os.Stderr, "Batch processing interrupted. Skipping remaining %d download(s).\n", len(urls)-i)
			for j := i; j < len(urls); j++ {
				failure = append(failure, fmt.Sprintf("%s (skipped due to interruption)", urls[j]))
			}
			break
		}

		fmt.Println("→ Downloading:", url)

		var cmdArgs []string
		if absTargetDir != "" {
			cmdArgs = append(cmdArgs, "-d", absTargetDir, url)
		} else {
			cmdArgs = append(cmdArgs, url)
		}

		cmd := exec.Command("dlfast", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()

		if err != nil {
			if exitErr, ok := err.(*exec.ExitError); ok {
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					if ws.Signaled() {
						failure = append(failure, fmt.Sprintf("%s (dlfast process terminated by signal: %s)", url, ws.Signal()))
					} else {
						if ws.ExitStatus() == 130 && mainCtx.Err() == context.Canceled {
							failure = append(failure, fmt.Sprintf("%s (download interrupted, dlfast exited 130)", url))
						} else {
							failure = append(failure, fmt.Sprintf("%s (dlfast process exited with code: %d)", url, ws.ExitStatus()))
						}
					}
				} else {
					failure = append(failure, fmt.Sprintf("%s (dlfast process exited with code: %d - no WaitStatus)", url, exitErr.ExitCode()))
				}
			} else {
				failure = append(failure, fmt.Sprintf("%s (error running dlfast process: %v)", url, err))
			}
		} else {
			success = append(success, url)
		}
	}

	signal.Stop(sigChan)
	close(sigChan)

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
	}

	if mainCtx.Err() == context.Canceled {
		fmt.Println("Batch processing was interrupted by signal.")
		os.Exit(130)
	} else {
		if len(failure) > 0 {
			os.Exit(1)
		} else {
			fmt.Println("All downloads processed successfully!")
			os.Exit(0)
		}
	}
}
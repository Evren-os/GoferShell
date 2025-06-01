package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
)

const individualDownloadTimeout = 3 * time.Hour

func main() {
	if _, err := exec.LookPath("dlfast"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: 'dlfast' executable not found in PATH. Please ensure it is correctly installed.")
		os.Exit(1)
	}

	var targetDirFlag string
	flag.StringVar(&targetDirFlag, "d", "", "Target directory for all downloads. If not provided, dlfast uses its default (current directory).")

	flag.Usage = func() {
		cmdName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "%s: Download multiple files in batch using 'dlfast'.\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Usage: %s [-d target_directory] <URL1> [URL2 ...]\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Arguments:\n")
		fmt.Fprintf(os.Stderr, "  URL1 [URL2 ...]    One or more URLs to download.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExample:\n")
		fmt.Fprintf(os.Stderr, "  %s -d /path/to/downloads \"http://example.com/file1.zip\" \"http://example.com/file2.tar.gz\"\n", cmdName)
	}

	flag.Parse()

	urls := flag.Args()
	if len(urls) == 0 {
		flag.Usage()
		os.Exit(1)
	}

	var absTargetDir string
	if targetDirFlag != "" {
		var err error
		absTargetDir, err = filepath.Abs(targetDirFlag)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not resolve target directory '%s': %v\n", targetDirFlag, err)
			os.Exit(1)
		}
		if err := os.MkdirAll(absTargetDir, 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not create target directory '%s': %v\n", absTargetDir, err)
			os.Exit(1)
		}
		info, err := os.Stat(absTargetDir)
		if err != nil || !info.IsDir() {
			fmt.Fprintf(os.Stderr, "Error: Target path '%s' is not a valid directory.\n", absTargetDir)
			os.Exit(1)
		}
		fmt.Printf("Batch download target directory: %s\n", absTargetDir)
	} else {
		cwd, _ := os.Getwd()
		fmt.Printf("Batch download target directory: Not specified, dlfast will use its default (typically current directory: %s)\n", cwd)
	}

	mainCtx, mainCancel := context.WithCancel(context.Background())
	defer mainCancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Fprintf(os.Stderr, "\nSignal (%s) received by dlfast_batch, attempting to stop all downloads...\n", sig)
			mainCancel()
		case <-mainCtx.Done():
			return
		}
	}()

	var success, failure []string
	fmt.Printf("\nStarting batch download of %d URL(s)...\n", len(urls))

	for i, url := range urls {
		fmt.Printf("\n[%d/%d] Processing URL: %s\n", i+1, len(urls), url)

		if mainCtx.Err() != nil {
			fmt.Fprintf(os.Stderr, "Batch processing interrupted. Skipping remaining %d download(s).\n", len(urls)-i)
			for j := i; j < len(urls); j++ {
				failure = append(failure, fmt.Sprintf("%s (skipped due to batch interruption)", urls[j]))
			}
			break
		}

		var cmdArgs []string
		if absTargetDir != "" {
			cmdArgs = append(cmdArgs, "-d", absTargetDir, url)
		} else {
			cmdArgs = append(cmdArgs, url)
		}

		dlCtx, dlCancel := context.WithTimeout(mainCtx, individualDownloadTimeout)

		cmd := exec.CommandContext(dlCtx, "dlfast", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr

		err := cmd.Run()
		dlCancel()

		if err != nil {
			if errors.Is(dlCtx.Err(), context.DeadlineExceeded) {
				failure = append(failure, fmt.Sprintf("%s (failed: download timed out after %v)", url, individualDownloadTimeout))
			} else if errors.Is(dlCtx.Err(), context.Canceled) {
				failure = append(failure, fmt.Sprintf("%s (failed: download cancelled as part of batch interruption)", url))
			} else if exitErr, ok := err.(*exec.ExitError); ok {
				if ws, ok := exitErr.Sys().(syscall.WaitStatus); ok && ws.Signaled() {
					failure = append(failure, fmt.Sprintf("%s (failed: dlfast process terminated by signal %s)", url, ws.Signal()))
				} else {
					failure = append(failure, fmt.Sprintf("%s (failed: dlfast exited with code %d)", url, exitErr.ExitCode()))
				}
			} else {
				failure = append(failure, fmt.Sprintf("%s (failed: error running dlfast: %v)", url, err))
			}
		} else {
			success = append(success, url)
		}
	}

	signal.Stop(sigChan)

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

	if mainCtx.Err() == context.Canceled && len(success) < len(urls) {
		fmt.Println("\nBatch processing was interrupted by signal.")
		os.Exit(130)
	} else if len(failure) > 0 {
		fmt.Println("\nBatch completed with one or more failures.")
		os.Exit(1)
	} else {
		fmt.Println("\nAll downloads in batch processed successfully!")
		os.Exit(0)
	}
}

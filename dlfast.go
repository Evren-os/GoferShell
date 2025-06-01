package main

import (
	"context"
	"flag"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"
)

func inferFilenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("download_error_parsing_url_%s", time.Now().Format("20060102150405"))
	}

	p := u.Path
	if strings.HasSuffix(p, "/") && len(p) > 1 {
		p = p[:len(p)-1]
	}
	filename := filepath.Base(p)

	if filename == "" || filename == "." || filename == "/" {
		if u.Host != "" {
			name := strings.ReplaceAll(u.Host, ".", "_")
			name = strings.ReplaceAll(name, ":", "_")
			// Basic sanitization
			name = strings.Map(func(r rune) rune {
				if r == '/' || r == '\\' || r == ':' || r == '*' || r == '?' || r == '"' || r == '<' || r == '>' || r == '|' {
					return '_'
				}
				return r
			}, name)
			return fmt.Sprintf("download_from_%s_%s", name, time.Now().Format("150405"))
		}
		return fmt.Sprintf("downloaded_file_%s", time.Now().Format("20060102_150405"))
	}
	return filename
}

func main() {
	var destination string
	flag.StringVar(&destination, "d", "", "Target directory or full filepath. If empty, downloads to current directory with inferred filename.")

	flag.Usage = func() {
		cmdName := filepath.Base(os.Args[0])
		fmt.Fprintf(os.Stderr, "dlfast: A CLI tool to download files using aria2c.\n\n")
		fmt.Fprintf(os.Stderr, "Usage: %s [-d target_directory_or_filepath] <URL>\n\n", cmdName)
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  %s http://example.com/file.zip\n", cmdName)
		fmt.Fprintf(os.Stderr, "     (Downloads to current directory as 'file.zip' or similar inferred name)\n")
		fmt.Fprintf(os.Stderr, "  %s -d /mnt/data/ http://example.com/file.zip\n", cmdName)
		fmt.Fprintf(os.Stderr, "     (Downloads to /mnt/data/file.zip or similar inferred name)\n")
		fmt.Fprintf(os.Stderr, "  %s -d ~/Downloads/archive.zip http://example.com/file.zip\n", cmdName)
		fmt.Fprintf(os.Stderr, "     (Downloads to ~/Downloads/archive.zip)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() != 1 {
		flag.Usage()
		os.Exit(1)
	}
	downloadURL := flag.Arg(0)

	var targetDir, outputFilename string

	if destination == "" {
		cwd, err := os.Getwd()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not get current working directory: %v\n", err)
			os.Exit(1)
		}
		targetDir = cwd
		outputFilename = inferFilenameFromURL(downloadURL)
		fmt.Printf("Output directory: %s (current), Filename: %s (inferred)\n", targetDir, outputFilename)
	} else {
		absDest, err := filepath.Abs(destination)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: Could not resolve destination path '%s': %v\n", destination, err)
			os.Exit(1)
		}

		info, statErr := os.Stat(absDest)
		isLikelyDir := (statErr == nil && info.IsDir()) || strings.HasSuffix(destination, string(filepath.Separator))

		if isLikelyDir {
			targetDir = absDest
			outputFilename = inferFilenameFromURL(downloadURL)
			fmt.Printf("Output directory: %s, Filename: %s (inferred)\n", targetDir, outputFilename)
		} else {
			targetDir = filepath.Dir(absDest)
			outputFilename = filepath.Base(absDest)
			if outputFilename == "" || outputFilename == "." || outputFilename == string(filepath.Separator) {
				fmt.Fprintf(os.Stderr, "Error: Invalid filename '%s' derived from path '%s'\n", outputFilename, absDest)
				os.Exit(1)
			}
			fmt.Printf("Output file: %s\n", absDest)
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error: Could not create directory '%s': %v\n", targetDir, err)
		os.Exit(1)
	}

	tmpTestFile, err := os.CreateTemp(targetDir, ".dlfast-write-check-")
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: Directory '%s' does not appear to be writable: %v\n", targetDir, err)
		os.Exit(1)
	}
	tmpTestFile.Close()
	os.Remove(tmpTestFile.Name())

	aria2cArgs := []string{
		"--dir", targetDir,
		"--out", outputFilename,
		"--continue=true",
		"--max-connection-per-server=16",
		"--split=16",
		"--min-split-size=1M",
		"--file-allocation=falloc",
		"--max-tries=5",
		"--retry-wait=10",
		"--connect-timeout=30",
		"--timeout=60",
		"--max-file-not-found=3",
		"--summary-interval=1",
		"--console-log-level=warn",
		"--auto-file-renaming=false",
		"--allow-overwrite=true",
		"--conditional-get=true",
		"--check-integrity=true",
		"--disk-cache=64M",
		"--async-dns=true",
		"--http-accept-gzip=true",
		"--remote-time=true",
		downloadURL,
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigChan:
			fmt.Fprintf(os.Stderr, "\nSignal (%s) received by dlfast, cancelling download for %s...\n", sig, downloadURL)
			cancel()
		case <-ctx.Done():
			return
		}
	}()

	fullOutputPath := filepath.Join(targetDir, outputFilename)
	fmt.Printf("Starting download: %s -> %s\n", downloadURL, fullOutputPath)

	cmd := exec.CommandContext(ctx, "aria2c", aria2cArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err = cmd.Run()

	signal.Stop(sigChan)

	if err != nil {
		if ctx.Err() == context.Canceled {
			fmt.Fprintf(os.Stderr, "Download for %s was cancelled.\n", downloadURL)
			os.Exit(130)
		} else if exitErr, ok := err.(*exec.ExitError); ok {
			fmt.Fprintf(os.Stderr, "aria2c failed for %s. Exit code: %d\n", downloadURL, exitErr.ExitCode())
			os.Exit(exitErr.ExitCode())
		} else {
			fmt.Fprintf(os.Stderr, "Error executing aria2c for %s: %v\n", downloadURL, err)
			os.Exit(1)
		}
	} else {
		fmt.Printf("\nDownload completed successfully: %s\n", fullOutputPath)
	}
}

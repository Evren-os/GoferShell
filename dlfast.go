package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"
)

// Pre-compiled regex patterns for better performance
var (
	contentDispositionFilenameStarRe = regexp.MustCompile(`filename\*\s*=\s*([^;]+)`)
	contentDispositionFilenameRe     = regexp.MustCompile(`filename\s*=\s*([^;]+)`)
	dangerousCharsRe                 = regexp.MustCompile(`[<>:"/\\|?*]`)
)

const (
	maxConnectionsPerServer  = 16
	defaultParallelDownloads = 3
	defaultTimeout           = 60
	defaultConnectTimeout    = 30
	defaultMaxTries          = 5
	defaultRetryWait         = 10
)

type Config struct {
	Destination       string
	MaxSpeed          string
	Timeout           int
	ConnectTimeout    int
	MaxTries          int
	RetryWait         int
	UserAgent         string
	ParallelDownloads int
	Quiet             bool
}

type DownloadItem struct {
	URL      string
	Filename string
	FilePath string
	Error    error
}

// detectFilename makes an HTTP HEAD request to determine the actual filename
func detectFilename(ctx context.Context, rawURL, userAgent string, timeout int) (string, error) {
	client := &http.Client{
		Timeout: time.Duration(timeout) * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("too many redirects")
			}
			return nil
		},
	}

	req, err := http.NewRequestWithContext(ctx, "HEAD", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	} else {
		req.Header.Set("User-Agent", "dlfast/1.0")
	}

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP HEAD request: %w", err)
	}
	defer resp.Body.Close()

	// Try Content-Disposition header first
	if filename := parseContentDisposition(resp.Header.Get("Content-Disposition")); filename != "" {
		return sanitizeFilename(filename), nil
	}

	// Fallback to URL-based filename
	return inferFilenameFromURL(rawURL), nil
}

// parseContentDisposition parses RFC 6266 Content-Disposition header
func parseContentDisposition(header string) string {
	if header == "" {
		return ""
	}

	// Look for filename* parameter (RFC 5987 encoded)
	if matches := contentDispositionFilenameStarRe.FindStringSubmatch(header); len(matches) > 1 {
		encoded := strings.Trim(matches[1], `"' `)
		if decoded := decodeRFC5987(encoded); decoded != "" {
			return decoded
		}
	}

	// Look for regular filename parameter
	if matches := contentDispositionFilenameRe.FindStringSubmatch(header); len(matches) > 1 {
		filename := strings.Trim(matches[1], `"' `)
		return filename
	}

	return ""
}

// decodeRFC5987 decodes RFC 5987 encoded filenames
func decodeRFC5987(encoded string) string {
	parts := strings.SplitN(encoded, "'", 3)
	if len(parts) != 3 {
		return ""
	}

	// Simple URL decode for the filename part
	decoded, err := url.QueryUnescape(parts[2])
	if err != nil {
		return ""
	}

	return decoded
}

// sanitizeFilename removes or replaces dangerous characters
func sanitizeFilename(filename string) string {
	// Remove or replace dangerous characters
	filename = dangerousCharsRe.ReplaceAllString(filename, "_")

	// Remove leading/trailing spaces and dots
	filename = strings.Trim(filename, " .")

	// Ensure it's not empty and not a reserved name
	if filename == "" || isReservedName(filename) {
		return fmt.Sprintf("download_%s", time.Now().Format("20060102_150405"))
	}

	return filename
}

// isReservedName checks for Windows reserved filenames
func isReservedName(name string) bool {
	reserved := []string{"CON", "PRN", "AUX", "NUL", "COM1", "COM2", "COM3", "COM4",
		"COM5", "COM6", "COM7", "COM8", "COM9", "LPT1", "LPT2", "LPT3", "LPT4",
		"LPT5", "LPT6", "LPT7", "LPT8", "LPT9"}

	upper := strings.ToUpper(name)
	for _, res := range reserved {
		if upper == res {
			return true
		}
	}
	return false
}

// inferFilenameFromURL extracts filename from URL (fallback method)
func inferFilenameFromURL(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Sprintf("download_error_%s", time.Now().Format("20060102150405"))
	}

	path := u.Path
	if strings.HasSuffix(path, "/") && len(path) > 1 {
		path = path[:len(path)-1]
	}
	filename := filepath.Base(path)

	if filename == "" || filename == "." || filename == "/" {
		if u.Host != "" {
			name := sanitizeFilename(u.Host)
			return fmt.Sprintf("download_from_%s_%s", name, time.Now().Format("150405"))
		}
		return fmt.Sprintf("downloaded_file_%s", time.Now().Format("20060102_150405"))
	}

	return sanitizeFilename(filename)
}

// buildAria2cArgs constructs optimized aria2c arguments
func buildAria2cArgs(targetDir, filename, url string, config *Config) []string {
	args := []string{
		"--dir=" + targetDir,
		"--out=" + filename,
		"--continue=true",
		"--max-connection-per-server=" + strconv.Itoa(maxConnectionsPerServer),
		"--split=32",
		"--min-split-size=1M",
		"--file-allocation=falloc",
		"--max-tries=" + strconv.Itoa(config.MaxTries),
		"--retry-wait=" + strconv.Itoa(config.RetryWait),
		"--connect-timeout=" + strconv.Itoa(config.ConnectTimeout),
		"--timeout=" + strconv.Itoa(config.Timeout),
		"--max-file-not-found=3",
		"--summary-interval=1",
		"--console-log-level=warn",
		"--auto-file-renaming=false",
		"--allow-overwrite=true",
		"--conditional-get=true",
		"--check-integrity=true",
		"--disk-cache=128M",
		"--async-dns=true",
		"--http-accept-gzip=true",
		"--remote-time=true",
	}

	if config.MaxSpeed != "" {
		args = append(args, "--max-download-limit="+config.MaxSpeed)
	}

	if config.UserAgent != "" {
		args = append(args, "--user-agent="+config.UserAgent)
	}

	args = append(args, url)
	return args
}

// validateURL performs comprehensive URL validation
func validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("URL cannot be empty")
	}

	u, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid URL format: %w", err)
	}

	if u.Scheme != "http" && u.Scheme != "https" && u.Scheme != "ftp" {
		return fmt.Errorf("unsupported URL scheme: %s (supported: http, https, ftp)", u.Scheme)
	}

	if u.Host == "" {
		return errors.New("URL must contain a host")
	}

	return nil
}

// setupDestination determines target directory and creates it if necessary
func setupDestination(destination string) (string, error) {
	var targetDir string

	if destination == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("getting current directory: %w", err)
		}
		targetDir = cwd
	} else {
		absDest, err := filepath.Abs(destination)
		if err != nil {
			return "", fmt.Errorf("resolving destination path '%s': %w", destination, err)
		}

		info, statErr := os.Stat(absDest)
		isDir := (statErr == nil && info.IsDir()) || strings.HasSuffix(destination, string(filepath.Separator))

		if isDir {
			targetDir = absDest
		} else {
			return "", fmt.Errorf("destination must be a directory, got: %s", destination)
		}
	}

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", fmt.Errorf("creating directory '%s': %w", targetDir, err)
	}

	// Test write permissions
	tmpFile, err := os.CreateTemp(targetDir, ".dlfast-write-check-")
	if err != nil {
		return "", fmt.Errorf("directory '%s' is not writable: %w", targetDir, err)
	}
	tmpFile.Close()
	os.Remove(tmpFile.Name())

	return targetDir, nil
}

// downloadFile performs a single download with aria2c
func downloadFile(ctx context.Context, item *DownloadItem, targetDir string, config *Config) error {
	if !config.Quiet {
		fmt.Printf("🔍 Detecting filename for: %s%s%s\n", colorCyan, item.URL, colorReset)
	}

	// Detect actual filename
	filename, err := detectFilename(ctx, item.URL, config.UserAgent, config.ConnectTimeout)
	if err != nil {
		if !config.Quiet {
			fmt.Printf("%s⚠️  Could not detect filename, using URL fallback: %v%s\n", colorYellow, err, colorReset)
		}
		// Fallback to URL-based inference on error
		filename = inferFilenameFromURL(item.URL)
	}

	item.Filename = filename
	item.FilePath = filepath.Join(targetDir, filename)

	if !config.Quiet {
		fmt.Printf("📥 Downloading: %s%s%s → %s%s%s\n", colorCyan, item.URL, colorReset, colorCyan, item.FilePath, colorReset)
	}

	args := buildAria2cArgs(targetDir, filename, item.URL, config)

	cmd := exec.CommandContext(ctx, "aria2c", args...)

	// Create new process group for proper signal handling
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
	}

	// Let aria2c output directly to terminal (unless quiet mode)
	if !config.Quiet {
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	} else {
		// In quiet mode, capture stderr for error reporting
		cmd.Stderr = os.Stderr
	}

	err = cmd.Run()

	if err != nil {
		if ctx.Err() == context.Canceled {
			// Kill process group on cancellation
			if cmd.Process != nil {
				syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
			}
			return ctx.Err()
		}
		// aria2c error codes: https://aria2.github.io/manual/en/html/aria2c.html#exit-status
		if exitErr, ok := err.(*exec.ExitError); ok {
			switch exitErr.ExitCode() {
			case 3:
				return fmt.Errorf("file not found or access denied")
			case 9:
				return fmt.Errorf("not enough disk space available")
			case 28:
				return fmt.Errorf("network timeout or connection refused")
			default:
				return fmt.Errorf("aria2c failed with exit code %d", exitErr.ExitCode())
			}
		}
		return fmt.Errorf("aria2c execution failed: %w", err)
	}

	if !config.Quiet {
		fmt.Printf("%s✅ Completed: %s%s\n", colorGreen, item.FilePath, colorReset)
	}

	return nil
}

// runDownloads orchestrates single or batch downloads
func runDownloads(ctx context.Context, urls []string, config *Config) error {
	targetDir, err := setupDestination(config.Destination)
	if err != nil {
		return err
	}

	// Validate all URLs first
	for _, url := range urls {
		if err := validateURL(url); err != nil {
			return fmt.Errorf("invalid URL '%s': %w", url, err)
		}
	}

	// Initialize downloads
	downloads := make([]DownloadItem, len(urls))
	for i, url := range urls {
		downloads[i] = DownloadItem{
			URL: url,
		}
	}

	if !config.Quiet {
		if len(urls) == 1 {
			fmt.Printf("Starting download...\n")
		} else {
			fmt.Printf("Starting batch download of %s%d%s files...\n", colorCyan, len(urls), colorReset)
		}
	}

	// Download coordination
	sem := make(chan struct{}, config.ParallelDownloads)
	var wg sync.WaitGroup
	errChan := make(chan error, len(urls))

	for i := range downloads {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()

			sem <- struct{}{}        // Acquire semaphore
			defer func() { <-sem }() // Release semaphore

			if !config.Quiet && len(urls) > 1 {
				fmt.Printf("\n[%s%d%s/%s%d%s] ", colorCyan, index+1, colorReset, colorCyan, len(urls), colorReset)
			}

			if err := downloadFile(ctx, &downloads[index], targetDir, config); err != nil {
				if errors.Is(err, context.Canceled) {
					if !config.Quiet {
						fmt.Printf("%s❌ Cancelled: %s%s\n", colorRed, downloads[index].URL, colorReset)
					}
				} else {
					if !config.Quiet {
						fmt.Printf("%s❌ Failed: %s - %v%s\n", colorRed, downloads[index].URL, err, colorReset)
					}
					errChan <- fmt.Errorf("download %d failed: %w", index+1, err)
				}
				return
			}
		}(i)
	}

	// Wait for all downloads
	wg.Wait()
	close(errChan)

	// Check for errors
	var downloadErrors []error
	for err := range errChan {
		downloadErrors = append(downloadErrors, err)
	}

	if ctx.Err() == context.Canceled {
		return fmt.Errorf("downloads cancelled by user")
	}

	if len(downloadErrors) > 0 {
		return fmt.Errorf("some downloads failed: %v", downloadErrors)
	}

	return nil
}

func main() {
	config := &Config{
		Timeout:           defaultTimeout,
		ConnectTimeout:    defaultConnectTimeout,
		MaxTries:          defaultMaxTries,
		RetryWait:         defaultRetryWait,
		ParallelDownloads: defaultParallelDownloads,
	}

	flag.StringVar(&config.Destination, "d", "", "Target directory for downloads")
	flag.StringVar(&config.MaxSpeed, "max-speed", "", "Maximum download speed (e.g., 1M, 500K)")
	flag.IntVar(&config.Timeout, "timeout", defaultTimeout, "Download timeout in seconds")
	flag.IntVar(&config.ConnectTimeout, "connect-timeout", defaultConnectTimeout, "Connection timeout in seconds")
	flag.IntVar(&config.MaxTries, "max-tries", defaultMaxTries, "Maximum retry attempts")
	flag.IntVar(&config.RetryWait, "retry-wait", defaultRetryWait, "Wait time between retries in seconds")
	flag.StringVar(&config.UserAgent, "user-agent", "", "Custom User-Agent string")
	flag.IntVar(&config.ParallelDownloads, "parallel", defaultParallelDownloads, "Number of parallel downloads (batch mode)")
	flag.BoolVar(&config.Quiet, "quiet", false, "Suppress progress display")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "dlfast: High-performance download tool powered by aria2c\n\n")
		fmt.Fprintf(os.Stderr, "Usage: dlfast [options] <URL> [URL2 ...]\n\n")
		fmt.Fprintf(os.Stderr, "Examples:\n")
		fmt.Fprintf(os.Stderr, "  dlfast https://example.com/file.zip\n")
		fmt.Fprintf(os.Stderr, "  dlfast -d ~/Downloads https://example.com/file1.zip https://example.com/file2.tar.gz\n")
		fmt.Fprintf(os.Stderr, "  dlfast --max-speed 1M --parallel 2 url1 url2 url3\n")
		fmt.Fprintf(os.Stderr, "  dlfast --user-agent \"MyBot/1.0\" --timeout 120 https://example.com/large.iso\n\n")
		fmt.Fprintf(os.Stderr, "Features:\n")
		fmt.Fprintf(os.Stderr, "  • Intelligent filename detection via HTTP Content-Disposition headers\n")
		fmt.Fprintf(os.Stderr, "  • Parallel batch downloads with configurable concurrency\n")
		fmt.Fprintf(os.Stderr, "  • Optimized for high-speed downloads (16 connections, 32 splits)\n")
		fmt.Fprintf(os.Stderr, "  • Robust signal handling and error recovery\n")
		fmt.Fprintf(os.Stderr, "  • Resume support for interrupted downloads\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() == 0 {
		flag.Usage()
		os.Exit(1)
	}

	// Check for aria2c availability
	if _, err := exec.LookPath("aria2c"); err != nil {
		fmt.Fprintf(os.Stderr, "%sError: aria2c not found in PATH. Please install aria2c.%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	urls := flag.Args()

	// Setup signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-sigChan
		fmt.Fprintf(os.Stderr, "\n%sReceived interrupt signal, cancelling downloads...%s\n", colorYellow, colorReset)
		cancel()
	}()

	// Run downloads
	if err := runDownloads(ctx, urls, config); err != nil {
		if errors.Is(err, context.Canceled) {
			fmt.Fprintf(os.Stderr, "%sDownloads cancelled.%s\n", colorYellow, colorReset)
			os.Exit(130)
		}
		fmt.Fprintf(os.Stderr, "%sError: %v%s\n", colorRed, err, colorReset)
		os.Exit(1)
	}

	if !config.Quiet {
		if len(urls) == 1 {
			fmt.Printf("%sDownload completed successfully!%s\n", colorGreen, colorReset)
		} else {
			fmt.Printf("%sAll downloads completed successfully!%s\n", colorGreen, colorReset)
		}
	}
}
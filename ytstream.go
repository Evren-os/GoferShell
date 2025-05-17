package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

func main() {
	// Check for required dependencies
	if _, err := exec.LookPath("yt-dlp"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: yt-dlp not found in PATH")
		os.Exit(1)
	}
	if _, err := exec.LookPath("mpv"); err != nil {
		fmt.Fprintln(os.Stderr, "Error: mpv not found in PATH")
		os.Exit(1)
	}

	// Parse command-line flags
	maxRes := flag.Int("max-res", 2160, "Maximum resolution (e.g., 2160 for 4K)")
	codec := flag.String("codec", "av1", "Preferred codec (av1 or vp9)")
	flag.Parse()

	// Validate URL argument
	args := flag.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Usage: ytstream [options] URL")
		fmt.Fprintln(os.Stderr, "Options: --max-res RES (default 2160), --codec CODEC (default av1)")
		os.Exit(1)
	}
	url := args[0]

	// Validate codec
	if *codec != "av1" && *codec != "vp9" {
		fmt.Fprintln(os.Stderr, "Invalid codec preference:", *codec, ". Use av1 or vp9.")
		os.Exit(1)
	}

	// Construct yt-dlp format and sort strings
	formatString := fmt.Sprintf("bv*[height<=%d]+ba/bv*[height<=%d]", *maxRes, *maxRes)
		var sortString string
		if *codec == "av1" {
			sortString = "res,fps,vcodec:av01,vcodec:vp9.2,vcodec:vp9,vcodec:hev1,acodec:opus"
		} else {
			sortString = "res,fps,vcodec:vp9,vcodec:vp9.2,vcodec:av01,vcodec:hev1,acodec:opus"
		}

		// Fetch stream URL with yt-dlp
		cmd := exec.Command("yt-dlp", "--prefer-free-formats", "--format", formatString, "--format-sort", sortString, "--get-url", url)
		output, err := cmd.Output()
		if err != nil {
			fmt.Fprintln(os.Stderr, "Failed to get stream URL:", err)
			os.Exit(1)
		}

		// Extract and validate stream URL
		lines := strings.Split(string(output), "\n")
		if len(lines) == 0 || lines[0] == "" {
			fmt.Fprintln(os.Stderr, "Failed to get stream URL")
			os.Exit(1)
		}
		streamURL := strings.TrimSpace(lines[0])

		// Play stream with mpv
		mpvCmd := exec.Command("mpv", streamURL)
		mpvCmd.Stdout = os.Stdout
		mpvCmd.Stderr = os.Stderr
		if err := mpvCmd.Run(); err != nil {
			fmt.Fprintln(os.Stderr, "Failed to run mpv:", err)
			os.Exit(1)
		}
}

package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// checkYTDLP ensures yt-dlp is installed and available
func checkYTDLP() error {
	cmd := exec.Command("yt-dlp", "--version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yt-dlp is not installed or not found in PATH: %v", err)
	}
	return nil
}

func main() {
	// Define command-line flags
	var maxRes int
	var codecPref string
	flag.IntVar(&maxRes, "max-res", 2160, "Maximum resolution in pixels (e.g., 2160 for 4K)")
	flag.StringVar(&codecPref, "codec", "av1", "Preferred codec (av1 or vp9)")

	// Parse flags
	flag.Parse()

	// Validate remaining arguments (URL)
	args := flag.Args()
	if len(args) < 1 {
		fmt.Println("Error: URL is required")
		fmt.Println("Usage: ytmax [options] URL")
		fmt.Println("Options:")
		flag.PrintDefaults()
		os.Exit(1)
	}
	url := args[0]

	// Verify yt-dlp is available
	if err := checkYTDLP(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}

	// Common yt-dlp options from YT_OPTS
	ytOpts := []string{
		"--prefer-free-formats",
		"--format-sort-force",
		"--merge-output-format", "mkv",
		"--concurrent-fragments", "3",
		"--no-mtime",
		"--output", "%(title)s [%(id)s][%(height)sp][%(fps)sfps][%(vcodec)s][%(acodec)s].%(ext)s",
		"--external-downloader", "aria2c",
		"--external-downloader-args", "-x 16 -s 16 -k 1M",
	}

	// Validate max resolution
	if maxRes <= 0 {
		fmt.Println("Error: Maximum resolution must be a positive integer")
		os.Exit(1)
	}
	formatString := fmt.Sprintf("bv*[height<=%d]+ba/bv*[height<=%d]", maxRes, maxRes)

		// Validate and set sort string based on codec preference
		codecPref = strings.ToLower(codecPref)
		var sortString string
		switch codecPref {
			case "av1":
				sortString = "res,fps,vcodec:av01,vcodec:vp9.2,vcodec:vp9,vcodec:hev1,acodec:opus"
			case "vp9":
				sortString = "res,fps,vcodec:vp9,vcodec:vp9.2,vcodec:av01,vcodec:hev1,acodec:opus"
			default:
				fmt.Println("Error: Invalid codec preference. Use 'av1' or 'vp9'")
				os.Exit(1)
		}

		// Build the yt-dlp command
		cmdArgs := append(ytOpts,
				  "--format", formatString,
		    "--format-sort", sortString,
		    url,
		)

		// Execute yt-dlp
		cmd := exec.Command("yt-dlp", cmdArgs...)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		if err != nil {
			fmt.Printf("Error executing yt-dlp: %v\n", err)
			os.Exit(1)
		}
}

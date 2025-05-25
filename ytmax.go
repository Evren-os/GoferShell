package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// checkYTDLP ensures yt-dlp is installed and available in PATH.
func checkYTDLP() error {
	cmd := exec.Command("yt-dlp", "--version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("yt-dlp is not installed or not found in PATH: %w", err)
	}
	return nil
}

func main() {
	var codecPref string
	var destinationPath string

	// Usage message
	flag.Usage = func() {
		output := flag.CommandLine.Output()
		fmt.Fprintf(output, "Usage: ytmax [options] URL\n\n")
		fmt.Fprintf(output, "ytmax is a command-line tool to download videos (primarily from YouTube) using yt-dlp,\n")
		fmt.Fprintf(output, "with specific preferences for quality (up to 4K) and codecs.\n\n")
		fmt.Fprintf(output, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(output, "\nExamples:\n")
		fmt.Fprintf(output, "  ytmax --codec vp9 -d /mnt/videos https://www.youtube.com/watch?v=VIDEO_ID\n")
		fmt.Fprintf(output, "  ytmax -d /mnt/videos/my_special_video.mkv https://www.youtube.com/watch?v=VIDEO_ID\n")
		fmt.Fprintf(output, "  ytmax -d ./my_downloads/ https://www.youtube.com/watch?v=VIDEO_ID  (saves into ./my_downloads/ using default filename pattern)\n")
	}

	// Define command-line flags
	flag.StringVar(&codecPref, "codec", "av1", "Preferred video codec (av1 or vp9).")
	flag.StringVar(&destinationPath, "d", "", "Download destination. Can be a directory (e.g., '~/videos/', './downloads/') or a full file path (e.g., '~/videos/my_video.mkv'). If a directory is specified, videos are saved there using a default naming scheme. If not specified, downloads to the current directory.")

	flag.Parse()

	args := flag.Args()
	if len(args) < 1 {
		flag.Usage()
		os.Exit(1)
	}
	url := args[0]

	// Verify yt-dlp dependency
	if err := checkYTDLP(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Filename pattern
	filenamePattern := "%(title)s [%(id)s][%(height)sp][%(fps)sfps][%(vcodec)s][%(acodec)s].%(ext)s"

	var outputArgForYTDLP string
	if destinationPath == "" {
		outputArgForYTDLP = filenamePattern
	} else {
		isTargetLikelyDirectory := false
		if strings.HasSuffix(destinationPath, string(os.PathSeparator)) {
			isTargetLikelyDirectory = true
		} else {
			info, err := os.Stat(destinationPath)
			if err == nil && info.IsDir() {
				isTargetLikelyDirectory = true
			}
		}

		if isTargetLikelyDirectory {
			outputArgForYTDLP = filepath.Join(destinationPath, filenamePattern)
		} else {
			outputArgForYTDLP = destinationPath
		}
	}

	// Base yt-dlp command options
	ytOpts := []string{
		"--prefer-free-formats",
		"--format-sort-force",
		"--merge-output-format", "mkv",
		"--concurrent-fragments", "3",
		"--no-mtime",
		"--output", outputArgForYTDLP,
		"--external-downloader", "aria2c",
		"--external-downloader-args", "-x 16 -s 16 -k 1M",
	}

	maxHeight := 2160
	formatString := fmt.Sprintf("bv*[height<=%d]+ba/bv*[height<=%d]", maxHeight, maxHeight)

	codecPref = strings.ToLower(codecPref)
	var sortString string
	switch codecPref {
	case "av1":
		sortString = "res,fps,vcodec:av01,vcodec:vp9.2,vcodec:vp9,vcodec:hev1,acodec:opus"
	case "vp9":
		sortString = "res,fps,vcodec:vp9,vcodec:vp9.2,vcodec:av01,vcodec:hev1,acodec:opus"
	default:
		fmt.Fprintln(os.Stderr, "Error: Invalid codec preference. Use 'av1' or 'vp9'.")
		flag.Usage()
		os.Exit(1)
	}

	cmdArgs := append(ytOpts,
		"--format", formatString,
		"--format-sort", sortString,
		url,
	)

	cmd := exec.Command("yt-dlp", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error executing yt-dlp: %v\n", err)
		os.Exit(1)
	}
}
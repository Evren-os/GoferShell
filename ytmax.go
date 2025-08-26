package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Constants for yt-dlp arguments and settings.
const (
	defaultFilenamePattern = "%(title)s [%(id)s][%(height)sp][%(fps)sfps][%(vcodec)s][%(acodec)s].%(ext)s"
	defaultMergeFormat     = "mkv"
	codecAV1               = "av1"
	codecVP9               = "vp9"

	// Settings for social media compatibility.
	socmFormat      = "bv[vcodec^=avc]+ba[acodec^=mp4a]/b[vcodec^=avc]"
	socmMergeFormat = "mp4"
)

// fatalf prints a formatted error message to stderr and exits with status 1.
func fatalf(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
	os.Exit(1)
}

// checkDependencies ensures that all required command-line tools are installed and in the PATH.
func checkDependencies(cmds ...string) {
	for _, cmd := range cmds {
		if _, err := exec.LookPath(cmd); err != nil {
			fatalf("%s is not installed or not found in PATH", cmd)
		}
	}
}

// buildYTDLPArgs constructs the command-line arguments for yt-dlp based on user flags.
func buildYTDLPArgs(url, codecPref, destinationPath, cookiesFrom string, socm bool) []string {
	// Determine output template.
	outputTemplate := defaultFilenamePattern
	if destinationPath != "" {
		if info, err := os.Stat(destinationPath); err == nil && info.IsDir() {
			outputTemplate = filepath.Join(destinationPath, defaultFilenamePattern)
		} else {
			outputTemplate = destinationPath
		}
	}

	// Base arguments.
	args := []string{
		"--prefer-free-formats",
		"--format-sort-force",
		"--no-mtime",
		"--output", outputTemplate,
		"--external-downloader", "aria2c",
		"--external-downloader-args", "-x 16 -s 16 -k 1M",
	}

	if cookiesFrom != "" {
		args = append(args, "--cookies-from-browser", cookiesFrom)
	}

	if socm {
		// Social media compatibility settings override others.
		args = append(args,
			"--merge-output-format", socmMergeFormat,
			"--format", socmFormat,
		)
	} else {
		// Standard high-quality download settings.
		maxHeight := 2160
		formatString := fmt.Sprintf("bv*[height<=%d]+ba/bv*[height<=%d]", maxHeight, maxHeight)

		var sortString string
		switch strings.ToLower(codecPref) {
		case codecAV1:
			sortString = "res,fps,vcodec:av01,vcodec:vp9.2,vcodec:vp9,vcodec:hev1,acodec:opus"
		case codecVP9:
			sortString = "res,fps,vcodec:vp9,vcodec:vp9.2,vcodec:av01,vcodec:hev1,acodec:opus"
		default:
			fatalf("Invalid codec preference. Use '%s' or '%s'.", codecAV1, codecVP9)
		}

		args = append(args,
			"--merge-output-format", defaultMergeFormat,
			"--format", formatString,
			"--format-sort", sortString,
		)
	}

	// Finally, add the URL.
	args = append(args, url)
	return args
}

func main() {
	// Define command-line flags.
	var (
		codecPref       string
		destinationPath string
		cookiesFrom     string
		socm            bool
	)

	flag.StringVar(&codecPref, "codec", codecAV1, "Preferred video codec (av1 or vp9). Ignored if -socm is used.")
	flag.StringVar(&destinationPath, "d", "", "Download destination. Can be a directory or a full file path.")
	flag.StringVar(&cookiesFrom, "cookies-from", "", "Load cookies from the specified browser (e.g., firefox, chrome).")
	flag.BoolVar(&socm, "socm", false, "Optimize for social media compatibility (MP4, H.264/AAC).")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: ytmax [options] URL\n\n")
		fmt.Fprintf(out, "A wrapper for yt-dlp to download single videos with specific quality and codec preferences.\n\n")
		fmt.Fprintf(out, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(out, "\nExamples:\n")
		fmt.Fprintf(out, "  ytmax -codec vp9 -d /mnt/videos https://youtu.be/VIDEO_ID\n")
		fmt.Fprintf(out, "  ytmax --cookies-from firefox https://youtu.be/VIDEO_ID\n")
	}

	flag.Parse()

	// Check for URL argument.
	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	url := flag.Arg(0)

	// Verify dependencies.
	checkDependencies("yt-dlp", "aria2c")

	// Build and execute the command.
	cmdArgs := buildYTDLPArgs(url, codecPref, destinationPath, cookiesFrom, socm)
	cmd := exec.Command("yt-dlp", cmdArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// Constants for codec names.
const (
	codecAV1 = "av1"
	codecVP9 = "vp9"
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

func main() {
	// Define command-line flags.
	var (
		maxRes      int
		codec       string
		cookiesFrom string
	)

	flag.IntVar(&maxRes, "max-res", 2160, "Maximum video resolution (e.g., 1080, 2160).")
	flag.StringVar(&codec, "codec", codecAV1, "Preferred video codec (av1 or vp9).")
	flag.StringVar(&cookiesFrom, "cookies-from", "", "Load cookies from the specified browser (e.g., firefox, chrome).")

	flag.Usage = func() {
		out := flag.CommandLine.Output()
		fmt.Fprintf(out, "Usage: ytstream [options] URL\n\n")
		fmt.Fprintf(out, "A tool to stream videos directly to a media player using a yt-dlp to mpv pipe.\n\n")
		fmt.Fprintf(out, "Options:\n")
		flag.PrintDefaults()
	}

	flag.Parse()

	if flag.NArg() < 1 {
		flag.Usage()
		os.Exit(1)
	}
	url := flag.Arg(0)

	checkDependencies("yt-dlp", "mpv")

	formatString := fmt.Sprintf("bv*[height<=%d]+ba/bv*[height<=%d]", maxRes, maxRes)
	var sortString string
	switch strings.ToLower(codec) {
	case codecAV1:
		sortString = "res,fps,vcodec:av01,vcodec:vp9.2,vcodec:vp9,vcodec:hev1,acodec:opus"
	case codecVP9:
		sortString = "res,fps,vcodec:vp9,vcodec:vp9.2,vcodec:av01,vcodec:hev1,acodec:opus"
	default:
		fatalf("Invalid codec preference. Use '%s' or '%s'.", codecAV1, codecVP9)
	}

	ytdlpArgs := []string{
		"--prefer-free-formats",
		"--format", formatString,
		"--format-sort", sortString,
		"-o", "-", // Output to standard output
	}

	if cookiesFrom != "" {
		ytdlpArgs = append(ytdlpArgs, "--cookies-from-browser", cookiesFrom)
	}

	ytdlpArgs = append(ytdlpArgs, url)

	cmdYtdlp := exec.Command("yt-dlp", ytdlpArgs...)
	cmdMpv := exec.Command("mpv", "-")

	pipe, err := cmdYtdlp.StdoutPipe()
	if err != nil {
		fatalf("failed to create pipe: %v", err)
	}
	cmdMpv.Stdin = pipe

	cmdYtdlp.Stderr = os.Stderr
	cmdMpv.Stderr = os.Stderr

	if err := cmdYtdlp.Start(); err != nil {
		fatalf("failed to start yt-dlp: %v", err)
	}
	if err := cmdMpv.Start(); err != nil {
		fatalf("failed to start mpv: %v", err)
	}

	if err := cmdMpv.Wait(); err != nil {
		_ = cmdYtdlp.Process.Kill()
		os.Exit(1)
	}

	_ = cmdYtdlp.Wait()
}
package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"time"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorBlue   = "\033[34m"
	colorReset  = "\033[0m"
)

func main() {
	noVersion := flag.Bool("no-ver", false, "Strip version details from output")
	flag.Parse()

	aurHelper := detectAURHelper()

	if _, err := exec.LookPath("checkupdates"); err != nil {
		fmt.Println(colorRed + "checkupdates is MIA. Install 'pacman-contrib' or rot." + colorReset)
		os.Exit(1)
	}

	if aurHelper == "" {
		aurHelper = "paru"
	}
	if _, err := exec.LookPath(aurHelper); err != nil {
		fmt.Println(colorRed + aurHelper + " is missing. Please install it." + colorReset)
		os.Exit(1)
	}

	if dbSyncNeeded() {
		if err := syncDatabase(); err != nil {
			fmt.Println(colorRed + "Sync failed. Internet’s dead or mirrors hate you." + colorReset)
			os.Exit(1)
		}
	}

	officialUpdates := fetchOfficialUpdates()
	aurUpdates := fetchAURUpdates(aurHelper)

	if *noVersion {
		officialUpdates = stripVersions(officialUpdates)
		aurUpdates = stripVersions(aurUpdates)
	}

	displayResults(officialUpdates, aurUpdates, aurHelper)
}

// Checks for common AUR helpers
func detectAURHelper() string {
	if _, err := exec.LookPath("paru"); err == nil {
		return "paru"
	}
	if _, err := exec.LookPath("yay"); err == nil {
		return "yay"
	}
	return ""
}

// Checks if the pacman sync databases are older than 24 hours
func dbSyncNeeded() bool {
	syncDir := "/var/lib/pacman/sync"
	files, err := os.ReadDir(syncDir)
	if err != nil {
		return true
	}
	for _, file := range files {
		if file.IsDir() {
			continue
		}
		info, err := file.Info()
		if err != nil {
			continue
		}
		if time.Since(info.ModTime()) > 24*time.Hour {
			return true
		}
	}
	return false
}

func syncDatabase() error {
	cmd := exec.Command("sudo", "pacman", "-Sy", "--quiet", "--noconfirm")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run()
}

// List pending official repository updates
func fetchOfficialUpdates() string {
	cmd := exec.Command("checkupdates")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return string(output)
}

// List pending AUR updates
func fetchAURUpdates(aurHelper string) string {
	cmd := exec.Command(aurHelper, "-Qua")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	lines := strings.Split(string(output), "\n")
	var filtered []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = strings.Join(strings.Fields(line), " ")
		if !strings.HasSuffix(line, "[ignored]") {
			filtered = append(filtered, line)
		}
	}
	return strings.Join(filtered, "\n")
}

func stripVersions(updates string) string {
	lines := strings.Split(updates, "\n")
	var packages []string
	for _, line := range lines {
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			packages = append(packages, parts[0])
		}
	}
	return strings.Join(packages, "\n")
}

func displayResults(official, aur, aurHelper string) {
	officialLines := strings.Split(strings.TrimSpace(official), "\n")
	aurLines := strings.Split(strings.TrimSpace(aur), "\n")
	hasOfficial := len(officialLines) > 0 && officialLines[0] != ""
	hasAur := len(aurLines) > 0 && aurLines[0] != ""

	if !hasOfficial && !hasAur {
		fmt.Println(colorBlue + "No updates. Your system mocks entropy." + colorReset)
		return
	}

	if hasOfficial {
		fmt.Printf("%s%d official updates. The grind never stops.%s\n", colorGreen, len(officialLines), colorReset)
		fmt.Println(official)
	} else {
		fmt.Println(colorBlue + "Official repos: barren." + colorReset)
	}

	if aurHelper != "" {
		if hasAur {
			fmt.Printf("%s%d AUR updates. They’re watching.%s\n", colorYellow, len(aurLines), colorReset)
			fmt.Println(aur)
		} else {
			fmt.Println(colorBlue + "AUR sleeps. Silence is deadly." + colorReset)
		}
	}
}

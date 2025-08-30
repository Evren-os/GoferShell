package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	colorRed    = "\033[31m"
	colorGreen  = "\033[32m"
	colorYellow = "\033[33m"
	colorCyan   = "\033[36m"
	colorReset  = "\033[0m"

	commandTimeout = 30 * time.Second
)

type updateResult struct {
	output string
	err    error
}

func main() {
	noVersion := flag.Bool("no-ver", false, "Strip version details from output")
	flag.Parse()

	// Verify required commands exist
	if _, err := exec.LookPath("checkupdates"); err != nil {
		fmt.Printf("%scheckupdates is MIA. Install 'pacman-contrib' or rot.%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	aurHelper := detectAURHelper()
	if aurHelper == "" {
		fmt.Printf("%sNo AUR helper found. Install paru or yay.%s\n", colorRed, colorReset)
		os.Exit(1)
	}

	// Fetch updates concurrently
	var wg sync.WaitGroup
	officialChan := make(chan updateResult, 1)
	aurChan := make(chan updateResult, 1)

	wg.Add(2)
	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				officialChan <- updateResult{"", fmt.Errorf("panic recovered: %v", r)}
			}
		}()
		output, err := fetchOfficialUpdates()
		officialChan <- updateResult{output, err}
	}()

	go func() {
		defer wg.Done()
		defer func() {
			if r := recover(); r != nil {
				aurChan <- updateResult{"", fmt.Errorf("panic recovered: %v", r)}
			}
		}()
		output, err := fetchAURUpdates(aurHelper)
		aurChan <- updateResult{output, err}
	}()

	wg.Wait()
	close(officialChan)
	close(aurChan)

	officialResult := <-officialChan
	aurResult := <-aurChan

	// Handle errors - only report actual failures, not "no updates"
	if officialResult.err != nil {
		fmt.Printf("%sFailed to check official updates: %v%s\n", colorRed, officialResult.err, colorReset)
		os.Exit(1)
	}
	if aurResult.err != nil {
		fmt.Printf("%sFailed to check AUR updates: %v%s\n", colorRed, aurResult.err, colorReset)
		os.Exit(1)
	}

	officialUpdates := officialResult.output
	aurUpdates := aurResult.output

	if *noVersion {
		officialUpdates = stripVersions(officialUpdates)
		aurUpdates = stripVersions(aurUpdates)
	}

	displayResults(officialUpdates, aurUpdates)
}

func detectAURHelper() string {
	helpers := []string{"paru", "yay"}
	for _, helper := range helpers {
		if _, err := exec.LookPath(helper); err == nil {
			return helper
		}
	}
	return ""
}

func runCommand(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out")
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func fetchOfficialUpdates() (string, error) {
	output, err := runCommand("checkupdates")
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 2 {
			return "", nil // Exit code 2 means no updates
		}
		return "", err
	}
	return output, nil
}

func fetchAURUpdates(aurHelper string) (string, error) {
	output, err := runCommand(aurHelper, "-Qua")
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return "", nil // Exit code 1 means no updates for paru/yay
		}
		return "", err
	}

	if output == "" {
		return "", nil
	}

	lines := strings.Split(output, "\n")
	var builder strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Normalize whitespace and filter ignored packages
		line = strings.Join(strings.Fields(line), " ")
		if !strings.HasSuffix(line, "[ignored]") {
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(line)
		}
	}

	return builder.String(), nil
}

func stripVersions(updates string) string {
	if updates == "" {
		return ""
	}

	lines := strings.Split(updates, "\n")
	var builder strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) > 0 {
			if builder.Len() > 0 {
				builder.WriteByte('\n')
			}
			builder.WriteString(parts[0])
		}
	}

	return builder.String()
}

func countUpdates(updates string) int {
	if updates == "" {
		return 0
	}
	count := 0
	for _, r := range updates {
		if r == '\n' {
			count++
		}
	}
	// Add 1 for the last line if the string is not empty
	if len(updates) > 0 {
		count++
	}
	return count
}

func displayResults(official, aur string) {
	officialCount := countUpdates(official)
	aurCount := countUpdates(aur)

	if officialCount == 0 && aurCount == 0 {
		fmt.Printf("%sAll patched. The universe is in balance.%s\n", colorGreen, colorReset)
		return
	}

	if officialCount > 0 {
		fmt.Printf("%sThe mothership is hailing: %s%d%s new directives.%s\n", colorGreen, colorCyan, officialCount, colorGreen, colorReset)
		fmt.Println(official)
	} else {
		fmt.Printf("%sMainline is stable. As it should be.%s\n", colorGreen, colorReset)
	}

	if aurCount > 0 {
		fmt.Printf("%s%s%d%s new AUR bounties.%s\n", colorYellow, colorCyan, aurCount, colorYellow, colorReset)
		fmt.Println(aur)
	} else {
		fmt.Printf("%sAUR sleeps. Silence is deadly.%s\n", colorGreen, colorReset)
	}
}

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
	colorBlue   = "\033[34m"
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
		output, err := fetchOfficialUpdates()
		officialChan <- updateResult{output, err}
	}()

	go func() {
		defer wg.Done()
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

func runCommandWithTimeout(name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), commandTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.Output()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("command timed out")
		}
		// Handle common "no updates" exit codes
		if exitErr, ok := err.(*exec.ExitError); ok {
			exitCode := exitErr.ExitCode()
			if exitCode == 1 || exitCode == 2 {
				return "", nil
			}
		}
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func fetchOfficialUpdates() (string, error) {
	return runCommandWithTimeout("checkupdates")
}

func fetchAURUpdates(aurHelper string) (string, error) {
	output, err := runCommandWithTimeout(aurHelper, "-Qua")
	if err != nil {
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
	
	lines := strings.Split(updates, "\n")
	count := 0
	for _, line := range lines {
		if strings.TrimSpace(line) != "" {
			count++
		}
	}
	return count
}

func displayResults(official, aur string) {
	officialCount := countUpdates(official)
	aurCount := countUpdates(aur)

	if officialCount == 0 && aurCount == 0 {
		fmt.Printf("%sAll patched. The universe is in balance.%s\n", colorBlue, colorReset)
		return
	}

	if officialCount > 0 {
		fmt.Printf("%sThe mothership is hailing: %d new directives.%s\n", colorGreen, officialCount, colorReset)
		fmt.Println(official)
	} else {
		fmt.Printf("%sMainline is stable. As it should be.%s\n", colorBlue, colorReset)
	}

	if aurCount > 0 {
		fmt.Printf("%s%d new AUR bounties.%s\n", colorYellow, aurCount, colorReset)
		fmt.Println(aur)
	} else {
		fmt.Printf("%sAUR sleeps. Silence is deadly.%s\n", colorBlue, colorReset)
	}
}
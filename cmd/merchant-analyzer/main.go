//go:generate go run ../gendoc ../../docs

// Package main is the entry point for the merchant-analyzer CLI.
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/checkers/aireadiness"
	"github.com/johlun99/merchant-analyzer/internal/checkers/attributes"
	googlespec "github.com/johlun99/merchant-analyzer/internal/checkers/googlespec"
	xmlchecker "github.com/johlun99/merchant-analyzer/internal/checkers/xml"
	"github.com/johlun99/merchant-analyzer/internal/exporter"
	"github.com/johlun99/merchant-analyzer/internal/feed"
	"github.com/johlun99/merchant-analyzer/internal/ui"
	"github.com/johlun99/merchant-analyzer/internal/ui/views"
)

var version = "dev" //nolint:gochecknoglobals

func main() {
	if err := rootCmd().Execute(); err != nil {
		os.Exit(1)
	}
}

func rootCmd() *cobra.Command {
	var outputFile string
	var noTUI bool

	cmd := &cobra.Command{
		Use:     "merchant-analyzer <url|file>",
		Short:   "Analyze a merchant product feed",
		Long:    "Fetch and analyze a merchant feed URL or local file, checking XML quality, attribute coverage, and AI readiness.",
		Args:    cobra.ExactArgs(1),
		Version: version,
		RunE: func(_ *cobra.Command, args []string) error {
			return run(args[0], outputFile, noTUI)
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "export report to file (.json or .md)")
	cmd.Flags().BoolVar(&noTUI, "no-tui", false, "print plain-text summary without interactive TUI")
	cmd.Flags().BoolP("version", "v", false, "print version and exit")

	return cmd
}

func loadFeed(source string) (*feed.Feed, error) {
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		return feed.Fetch(source)
	}
	return feed.FromFile(source)
}

const githubReleasesURL = "https://api.github.com/repos/johlun99/merchant-analyzer/releases/latest"

func fetchLatestVersion(timeout time.Duration) (string, bool) {
	return fetchLatestVersionFromURL(githubReleasesURL, timeout)
}

func fetchLatestVersionFromURL(url string, timeout time.Duration) (string, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil) //nolint:noctx
	if err != nil {
		return "", false
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := http.DefaultClient.Do(req) //nolint:gosec
	if err != nil || resp.StatusCode != http.StatusOK {
		return "", false
	}
	defer func() { _ = resp.Body.Close() }()

	var rel struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return "", false
	}
	return rel.TagName, rel.TagName != "" && rel.TagName != version
}

func run(source, outputFile string, noTUI bool) error {
	updateCh := make(chan string, 1)
	if version != "dev" {
		go func() {
			if latest, ok := fetchLatestVersion(5 * time.Second); ok {
				updateCh <- latest
			}
		}()
	}

	if strings.HasPrefix(source, "http") {
		fmt.Fprintf(os.Stderr, "Fetching %s...\n", source)
	} else {
		fmt.Fprintf(os.Stderr, "Reading %s...\n", source)
	}

	f, err := loadFeed(source)
	if err != nil {
		return fmt.Errorf("load feed: %w", err)
	}

	select {
	case latest := <-updateCh:
		fmt.Fprintf(os.Stderr, "\n  Update available: %s → %s\n", version, latest)
		fmt.Fprintf(os.Stderr, "  go install github.com/johlun99/merchant-analyzer/cmd/merchant-analyzer@latest\n\n")
	default:
	}

	checkers := []checker.Checker{
		xmlchecker.NewChecker(),
		attributes.NewChecker(),
		googlespec.NewChecker(),
		aireadiness.NewChecker(),
	}

	if noTUI || outputFile != "" {
		return runNoTUI(f, checkers, outputFile)
	}

	return runTUI(f, checkers)
}

func runTUI(f *feed.Feed, checkers []checker.Checker) error {
	m := ui.New(f, checkers)
	p := tea.NewProgram(m, tea.WithAltScreen())
	m.SetProgram(p)

	if _, err := p.Run(); err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}

func printProgress(done, total int, label string) {
	const width = 24
	filled := 0
	if total > 0 {
		filled = done * width / total
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)
	fmt.Fprintf(os.Stderr, "\r  [%s] %d/%d  %-30s", bar, done, total, label)
}

func runNoTUI(f *feed.Feed, checkers []checker.Checker, outputFile string) error {
	ctx := context.Background()
	total := len(checkers)
	var results []checker.Result
	for i, c := range checkers {
		if outputFile != "" {
			printProgress(i, total, c.Name()+"...")
		}
		results = append(results, c.Run(ctx, f))
	}
	if outputFile != "" {
		printProgress(total, total, "Done")
		fmt.Fprintln(os.Stderr)
	}

	fmt.Printf("\nmerchant-analyzer — %s\n", f.URL)
	fmt.Printf("Size: %s  Fetch: %dms  Products: %d\n\n",
		formatBytes(f.Size), f.FetchTime.Milliseconds(), f.ProductCount)

	hasError := false
	for _, r := range results {
		label := statusLabel(r.Status)
		extra := ""
		if r.Score != nil {
			extra = fmt.Sprintf(" — Score: %d/100", *r.Score)
		}
		if len(r.Items) > 0 && r.Items[0].Message != "" {
			extra += " — " + r.Items[0].Message
		}
		fmt.Printf("%s %s%s\n", label, r.Name, extra)
		if r.Status == checker.StatusError || r.Status == checker.StatusFatal {
			hasError = true
		}
	}
	fmt.Println()

	if outputFile != "" {
		report := views.BuildReport(f, results)
		if err := writeExport(report, outputFile); err != nil {
			return fmt.Errorf("export: %w", err)
		}
		fmt.Printf("Report exported to %s\n", outputFile)
	}

	if hasError {
		os.Exit(1)
	}
	return nil
}

func statusLabel(s checker.Status) string {
	switch s {
	case checker.StatusOK:
		return "[OK]     "
	case checker.StatusWarning:
		return "[WARNING]"
	case checker.StatusError:
		return "[ERROR]  "
	case checker.StatusFatal:
		return "[FATAL]  "
	}
	return "[FATAL]  "
}

func formatBytes(b int64) string {
	switch {
	case b >= 1024*1024:
		return fmt.Sprintf("%.1f MB", float64(b)/(1024*1024))
	case b >= 1024:
		return fmt.Sprintf("%.1f KB", float64(b)/1024)
	default:
		return fmt.Sprintf("%d B", b)
	}
}

func writeExport(report exporter.Report, filename string) error {
	ext := strings.ToLower(strings.TrimPrefix(filepath.Ext(filename), "."))
	switch ext {
	case "md":
		return os.WriteFile(filename, []byte(exporter.ToMarkdown(report)), 0o600)
	default:
		data, err := exporter.ToJSON(report)
		if err != nil {
			return err
		}
		return os.WriteFile(filename, data, 0o600)
	}
}

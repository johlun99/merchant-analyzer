// Package main generates the man page for merchant-analyzer.
// Run via: go generate ./...
package main

import (
	"log"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	outDir := "./docs"
	if len(os.Args) > 1 {
		outDir = os.Args[1]
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		log.Fatal(err)
	}
	cmd := buildCmd()
	header := &doc.GenManHeader{Title: "MERCHANT-ANALYZER", Section: "1"}
	if err := doc.GenManTree(cmd, header, outDir); err != nil {
		log.Fatal(err)
	}
}

// buildCmd returns a cobra command with the same structure as the real CLI,
// used solely for documentation generation.
func buildCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "merchant-analyzer <url|file>",
		Short: "Analyze a merchant product feed",
		Long:  "Fetch and analyze a merchant feed URL or local file, checking XML quality, attribute coverage, and AI readiness.",
	}
	cmd.Flags().StringP("output", "o", "", "export report to file (.json or .md)")
	cmd.Flags().Bool("no-tui", false, "print plain-text summary without interactive TUI")
	cmd.Flags().BoolP("version", "v", false, "print version and exit")
	return cmd
}

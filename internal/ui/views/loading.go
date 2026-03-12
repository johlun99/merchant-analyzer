// Package views provides Bubbletea view components for the merchant-analyzer TUI.
package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/ui/styles"
)

// CheckState tracks the status of a single checker in the loading view.
type CheckState struct {
	Name   string
	Done   bool
	Result *checker.Result
}

// LoadingView renders the loading screen with spinner and live checker list.
type LoadingView struct {
	URL     string
	Spinner spinner.Model
	Checks  []CheckState
	Width   int
}

// NewLoadingView creates a LoadingView with the given checker names.
func NewLoadingView(url string, names []string) LoadingView {
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(lipgloss.Color("#7C3AED"))

	checks := make([]CheckState, len(names))
	for i, n := range names {
		checks[i] = CheckState{Name: n}
	}
	return LoadingView{URL: url, Spinner: s, Checks: checks}
}

// MarkDone marks the checker with the given name as done.
func (v *LoadingView) MarkDone(result checker.Result) {
	for i := range v.Checks {
		if v.Checks[i].Name == result.Name {
			v.Checks[i].Done = true
			r := result
			v.Checks[i].Result = &r
			return
		}
	}
}

// View renders the loading screen.
func (v LoadingView) View() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n  %s\n\n", styles.StyleTitle.Render("merchant-analyzer"))
	fmt.Fprintf(&b, "  %s\n\n", styles.StyleSubtitle.Render("Analyzing: "+v.URL))

	for _, ch := range v.Checks {
		var icon, name string
		if !ch.Done {
			icon = v.Spinner.View()
			name = styles.StyleCheckPending.Render(ch.Name)
		} else {
			switch ch.Result.Status {
			case checker.StatusOK:
				icon = styles.StyleCheckOK.Render("✓")
				name = styles.StyleCheckOK.Render(ch.Name)
			case checker.StatusWarning:
				icon = styles.StyleCheckWarning.Render("⚠")
				name = styles.StyleCheckWarning.Render(ch.Name)
			case checker.StatusError, checker.StatusFatal:
				icon = styles.StyleCheckError.Render("✗")
				name = styles.StyleCheckError.Render(ch.Name)
			}
		}
		fmt.Fprintf(&b, "  %s  %s\n", icon, name)
	}

	fmt.Fprintf(&b, "\n  %s\n", styles.StyleHelp.Render("ctrl+c to abort"))
	return b.String()
}

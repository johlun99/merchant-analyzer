package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/ui/styles"
)

// ProductListView displays a full, searchable list of products affected by a single issue.
// No package-level vars are used (gochecknoglobals).
type ProductListView struct {
	CheckerName     string
	Field           string
	Message         string
	All             []checker.AffectedProduct
	Filtered        []checker.AffectedProduct
	Search          textinput.Model
	Viewport        viewport.Model
	Width           int
	Height          int
	ExportRequested bool
	ExportFilename  string
	exportOpen      bool
	exportInput     textinput.Model
	searchMode      bool
}

// NewProductListView creates an initialised ProductListView.
func NewProductListView(checkerName, field, message string, products []checker.AffectedProduct, w, h int) *ProductListView {
	ti := textinput.New()
	ti.Placeholder = "Search by ID or title..."
	ti.CharLimit = 100
	ti.Width = w - 18
	if ti.Width < 20 {
		ti.Width = 20
	}

	ei := textinput.New()
	ei.Placeholder = "filename.csv"
	ei.CharLimit = 100
	ei.Width = 32

	vp := viewport.New(w, h-7)

	v := &ProductListView{
		CheckerName: checkerName,
		Field:       field,
		Message:     message,
		All:         products,
		Filtered:    products,
		Search:      ti,
		exportInput: ei,
		Viewport:    vp,
		Width:       w,
		Height:      h,
	}
	v.refreshViewport()
	return v
}

// Update processes a message. Returns the updated view, an optional command,
// and done=true when the user wants to go back to the parent view.
func (v *ProductListView) Update(msg tea.Msg) (*ProductListView, tea.Cmd, bool) {
	keyMsg, isKey := msg.(tea.KeyMsg)
	if isKey {
		return v.handleKey(keyMsg)
	}
	if v.exportOpen {
		var cmd tea.Cmd
		v.exportInput, cmd = v.exportInput.Update(msg)
		return v, cmd, false
	}
	if v.searchMode {
		var cmd tea.Cmd
		v.Search, cmd = v.Search.Update(msg)
		v.filter(v.Search.Value())
		return v, cmd, false
	}
	var cmd tea.Cmd
	v.Viewport, cmd = v.Viewport.Update(msg)
	return v, cmd, false
}

func (v *ProductListView) handleKey(msg tea.KeyMsg) (*ProductListView, tea.Cmd, bool) {
	if v.exportOpen {
		return v.handleExportKey(msg)
	}
	switch msg.String() {
	case "esc":
		if v.searchMode {
			v.searchMode = false
			v.Search.Blur()
			return v, nil, false
		}
		return v, nil, true // signal exit to parent
	case "/":
		if !v.searchMode {
			v.searchMode = true
			return v, v.Search.Focus(), false
		}
	case "up", "k":
		if !v.searchMode {
			v.Viewport.LineUp(1)
			return v, nil, false
		}
	case "down", "j":
		if !v.searchMode {
			v.Viewport.LineDown(1)
			return v, nil, false
		}
	case "pgup":
		if !v.searchMode {
			v.Viewport.HalfViewUp()
			return v, nil, false
		}
	case "pgdown":
		if !v.searchMode {
			v.Viewport.HalfViewDown()
			return v, nil, false
		}
	case "e":
		if !v.searchMode {
			v.exportOpen = true
			v.exportInput.SetValue(strings.ReplaceAll(v.Field, " ", "_") + "-products.csv")
			return v, v.exportInput.Focus(), false
		}
	}
	if v.searchMode {
		var cmd tea.Cmd
		v.Search, cmd = v.Search.Update(msg)
		v.filter(v.Search.Value())
		return v, cmd, false
	}
	return v, nil, false
}

func (v *ProductListView) handleExportKey(msg tea.KeyMsg) (*ProductListView, tea.Cmd, bool) {
	switch msg.String() {
	case "esc":
		v.exportOpen = false
		v.exportInput.Blur()
		return v, nil, false
	case "enter":
		name := strings.TrimSpace(v.exportInput.Value())
		if name == "" {
			name = strings.ReplaceAll(v.Field, " ", "_") + "-products.csv"
		}
		v.ExportFilename = name
		v.ExportRequested = true
		v.exportOpen = false
		v.exportInput.Blur()
		return v, nil, false
	default:
		var cmd tea.Cmd
		v.exportInput, cmd = v.exportInput.Update(msg)
		return v, cmd, false
	}
}

// View renders the product list screen.
func (v *ProductListView) View() string {
	base := v.renderBase()
	if v.exportOpen {
		return v.renderWithOverlay(base)
	}
	return base
}

func (v *ProductListView) renderBase() string {
	var b strings.Builder

	// Breadcrumb
	crumb := fmt.Sprintf("  %s  ›  %s  —  %d products affected",
		v.CheckerName, v.Field, len(v.All))
	fmt.Fprintln(&b, styles.StyleTitle.Render(crumb))
	fmt.Fprintln(&b)

	// Search input
	searchLabel := styles.StyleMetricLabel.Render("  Search: ")
	fmt.Fprintf(&b, "%s%s\n\n", searchLabel, v.Search.View())

	// Product list viewport
	fmt.Fprint(&b, v.Viewport.View())

	// Help bar
	helpText := "  / search  ↑/↓ scroll  e export  esc back"
	if v.searchMode {
		helpText = "  type to filter  esc back  ↑/↓ scroll"
	}
	fmt.Fprintf(&b, "\n%s", styles.StyleHelp.Render(helpText))

	return b.String()
}

func (v *ProductListView) renderWithOverlay(base string) string {
	overlay := v.renderExportOverlay()
	overlayWidth := lipgloss.Width(overlay)
	startX := (v.Width - overlayWidth) / 2
	if startX < 0 {
		startX = 0
	}
	lines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	startY := len(lines) / 3
	if startY+len(overlayLines) > len(lines) {
		startY = 0
	}
	return placeOverlay(startX, startY, overlay, base)
}

func (v *ProductListView) renderExportOverlay() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styles.StyleTitle.Render("Export Product List"))
	count := fmt.Sprintf("%d products", len(v.Filtered))
	fmt.Fprintf(&b, "Exporting %s\n\n", styles.StyleMetric.Render(count))
	fmt.Fprintf(&b, "Filename: %s\n\n", v.exportInput.View())
	fmt.Fprintf(&b, "%s", styles.StyleHelp.Render("enter export  esc cancel"))
	return styles.StyleOverlay.Render(b.String())
}

func (v *ProductListView) filter(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		v.Filtered = v.All
	} else {
		filtered := make([]checker.AffectedProduct, 0, len(v.All))
		for _, p := range v.All {
			if strings.Contains(strings.ToLower(p.ID), query) ||
				strings.Contains(strings.ToLower(p.Title), query) ||
				strings.Contains(strings.ToLower(p.Value), query) {
				filtered = append(filtered, p)
			}
		}
		v.Filtered = filtered
	}
	v.Viewport.GotoTop()
	v.refreshViewport()
}

func (v *ProductListView) refreshViewport() {
	var b strings.Builder
	if len(v.Filtered) == 0 {
		fmt.Fprintf(&b, "  %s\n", styles.StyleMetricLabel.Render("No products match."))
	} else {
		for _, p := range v.Filtered {
			id := styles.StyleMetric.Render(p.ID)
			title := p.Title
			if title == "" {
				title = styles.StyleMetricLabel.Render("(no title)")
			}
			if p.Value != "" {
				val := styles.StyleMetricLabel.Render(p.Value)
				fmt.Fprintf(&b, "  %s  %s  %s\n", id, title, val)
			} else {
				fmt.Fprintf(&b, "  %s  %s\n", id, title)
			}
		}
	}
	v.Viewport.SetContent(b.String())
}

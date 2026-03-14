package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/ui/styles"
)

// ProductListView displays a full, searchable list of products affected by a single issue.
// No package-level vars are used (gochecknoglobals).
type ProductListView struct {
	CheckerName string
	Field       string
	All         []checker.AffectedProduct
	Filtered    []checker.AffectedProduct
	Search      textinput.Model
	Viewport    viewport.Model
	Width       int
	Height      int
	searchMode  bool
}

// NewProductListView creates an initialised ProductListView.
func NewProductListView(checkerName, field string, products []checker.AffectedProduct, w, h int) *ProductListView {
	ti := textinput.New()
	ti.Placeholder = "Search by ID or title..."
	ti.CharLimit = 100
	ti.Width = w - 18
	if ti.Width < 20 {
		ti.Width = 20
	}

	vp := viewport.New(w, h-7)

	v := &ProductListView{
		CheckerName: checkerName,
		Field:       field,
		All:         products,
		Filtered:    products,
		Search:      ti,
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
	switch msg.String() {
	case "esc":
		if v.searchMode {
			v.searchMode = false
			v.Search.Blur()
			v.Search.SetValue("")
			v.filter("")
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
	}
	if v.searchMode {
		var cmd tea.Cmd
		v.Search, cmd = v.Search.Update(msg)
		v.filter(v.Search.Value())
		return v, cmd, false
	}
	return v, nil, false
}

// View renders the product list screen.
func (v *ProductListView) View() string {
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
	helpText := "  / search  ↑/↓ scroll  esc back"
	if v.searchMode {
		helpText = "  type to filter  esc clear  ↑/↓ scroll"
	}
	fmt.Fprintf(&b, "\n%s", styles.StyleHelp.Render(helpText))

	return b.String()
}

func (v *ProductListView) filter(query string) {
	query = strings.ToLower(strings.TrimSpace(query))
	if query == "" {
		v.Filtered = v.All
	} else {
		filtered := make([]checker.AffectedProduct, 0, len(v.All))
		for _, p := range v.All {
			if strings.Contains(strings.ToLower(p.ID), query) ||
				strings.Contains(strings.ToLower(p.Title), query) {
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
			fmt.Fprintf(&b, "  %s  %s\n", id, title)
		}
	}
	v.Viewport.SetContent(b.String())
}

package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/exporter"
	"github.com/johlun99/merchant-analyzer/internal/feed"
	"github.com/johlun99/merchant-analyzer/internal/ui/styles"
)

// Tab indices.
const (
	TabOverview = iota
	TabXML
	TabChecks
	TabGoogleSpec
	TabAI
	TabAttributes
	TabCount
)

var tabNames = [TabCount]string{"Overview", "XML", "Checks", "Google Spec", "AI Score", "Attributes"}

// ReportView renders the tabbed report and export overlay.
type ReportView struct {
	Feed        *feed.Feed
	Results     []checker.Result
	ActiveTab   int
	Viewport    viewport.Model
	ExportOpen  bool
	ExportInput textinput.Model
	ExportMsg   string
	Width       int
	Height      int
}

// NewReportView creates an initialized ReportView.
func NewReportView(f *feed.Feed, results []checker.Result, width, height int) ReportView {
	vp := viewport.New(width, height-6)
	vp.SetContent("")

	ti := textinput.New()
	ti.Placeholder = "report.json or report.md"
	ti.CharLimit = 100
	ti.Width = 40

	rv := ReportView{
		Feed:        f,
		Results:     results,
		Viewport:    vp,
		ExportInput: ti,
		Width:       width,
		Height:      height,
	}
	rv.refreshViewport()
	return rv
}

// SetSize updates dimensions.
func (v *ReportView) SetSize(w, h int) {
	v.Width = w
	v.Height = h
	v.Viewport.Width = w
	v.Viewport.Height = h - 6
	v.refreshViewport()
}

// NextTab advances to the next tab.
func (v *ReportView) NextTab() {
	v.ActiveTab = (v.ActiveTab + 1) % TabCount
	v.Viewport.GotoTop()
	v.refreshViewport()
}

// PrevTab goes to the previous tab.
func (v *ReportView) PrevTab() {
	v.ActiveTab = (v.ActiveTab - 1 + TabCount) % TabCount
	v.Viewport.GotoTop()
	v.refreshViewport()
}

// OpenExport opens the export overlay.
func (v *ReportView) OpenExport() {
	v.ExportOpen = true
	v.ExportInput.Focus()
	v.ExportMsg = ""
}

// CloseExport closes the export overlay.
func (v *ReportView) CloseExport() {
	v.ExportOpen = false
	v.ExportInput.Blur()
	v.ExportInput.SetValue("")
}

// View renders the full report view including any overlay.
func (v ReportView) View() string {
	base := v.renderBase()
	if v.ExportOpen {
		return v.renderWithOverlay(base)
	}
	return base
}

func (v ReportView) renderBase() string {
	var b strings.Builder

	// Tab bar
	var tabs []string
	for i, name := range tabNames {
		if i == v.ActiveTab {
			tabs = append(tabs, styles.StyleTabActive.Render(name))
		} else {
			tabs = append(tabs, styles.StyleTabInactive.Render(name))
		}
	}
	tabBar := lipgloss.JoinHorizontal(lipgloss.Left, tabs...)
	fmt.Fprintln(&b, styles.StyleTabBar.Width(v.Width).Render(tabBar))

	// Viewport content
	fmt.Fprint(&b, v.Viewport.View())

	// Help bar
	fmt.Fprintf(&b, "\n%s", styles.StyleHelp.Render("  tab next  shift+tab prev  e export  q quit"))

	// Export success banner
	if v.ExportMsg != "" {
		fmt.Fprintf(&b, "  %s", styles.StyleSuccess.Render(v.ExportMsg))
	}

	return b.String()
}

func (v ReportView) renderWithOverlay(base string) string {
	overlay := v.renderExportOverlay()
	// Place overlay centered horizontally, ~1/3 from top
	lines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")
	overlayWidth := lipgloss.Width(overlay)
	startX := (v.Width - overlayWidth) / 2
	if startX < 0 {
		startX = 0
	}
	startY := len(lines) / 3
	if startY+len(overlayLines) > len(lines) {
		startY = 0
	}
	for i, ol := range overlayLines {
		y := startY + i
		if y < len(lines) {
			pad := strings.Repeat(" ", startX)
			lines[y] = pad + ol
		}
	}
	return strings.Join(lines, "\n")
}

func (v ReportView) renderExportOverlay() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styles.StyleTitle.Render("Export Report"))
	fmt.Fprintf(&b, "Filename: %s\n\n", v.ExportInput.View())
	fmt.Fprintf(&b, "%s", styles.StyleHelp.Render("enter confirm  esc cancel"))
	return styles.StyleOverlay.Render(b.String())
}

func (v *ReportView) refreshViewport() {
	switch v.ActiveTab {
	case TabOverview:
		v.Viewport.SetContent(v.renderOverview())
	case TabXML:
		v.Viewport.SetContent(v.renderCheckerTab("XML Validation"))
	case TabChecks:
		v.Viewport.SetContent(v.renderCheckerTab("Attribute Check"))
	case TabGoogleSpec:
		v.Viewport.SetContent(v.renderGoogleSpecTab())
	case TabAI:
		v.Viewport.SetContent(v.renderAITab())
	case TabAttributes:
		v.Viewport.SetContent(v.renderAttributesTab())
	}
}

func (v ReportView) renderOverview() string {
	var b strings.Builder

	fmt.Fprintf(&b, "\n")

	// Metrics row
	fmt.Fprintf(&b, "  %s  %s    %s  %s    %s  %s    %s  %s\n\n",
		styles.StyleMetricLabel.Render("URL"),
		styles.StyleMetric.Render(v.Feed.URL),
		styles.StyleMetricLabel.Render("Size"),
		styles.StyleMetric.Render(formatBytes(v.Feed.Size)),
		styles.StyleMetricLabel.Render("Fetch"),
		styles.StyleMetric.Render(fmt.Sprintf("%dms", v.Feed.FetchTime.Milliseconds())),
		styles.StyleMetricLabel.Render("Products"),
		styles.StyleMetric.Render(fmt.Sprintf("%d", v.Feed.ProductCount)),
	)

	// Checker summary
	for _, r := range v.Results {
		status := renderStatus(r.Status)
		score := ""
		if r.Score != nil {
			score = fmt.Sprintf("  Score: %s", styles.StyleMetric.Render(fmt.Sprintf("%d/100", *r.Score)))
		}
		issues := ""
		if len(r.Items) > 0 {
			issues = styles.StyleMetricLabel.Render(fmt.Sprintf("  (%d findings)", len(r.Items)))
		}
		fmt.Fprintf(&b, "  %s  %s%s%s\n", status, r.Name, score, issues)
	}

	return b.String()
}

func (v ReportView) renderCheckerTab(name string) string {
	var b strings.Builder
	for _, r := range v.Results {
		if r.Name != name {
			continue
		}
		fmt.Fprintf(&b, "\n  %s  %s\n\n", renderStatus(r.Status), r.Name)
		if len(r.Items) == 0 {
			fmt.Fprintf(&b, "  %s\n", styles.StyleCheckOK.Render("No issues found."))
			return b.String()
		}
		for _, item := range r.Items {
			bullet := styles.StyleMetricLabel.Render("  •")
			field := lipgloss.NewStyle().Bold(true).Render(item.Field)
			fmt.Fprintf(&b, "%s %s: %s\n", bullet, field, item.Message)
			for _, ex := range item.Examples {
				fmt.Fprintf(&b, "    %s\n", styles.StyleMetricLabel.Render(ex))
			}
		}
	}
	return b.String()
}

func (v ReportView) renderGoogleSpecTab() string {
	var b strings.Builder
	for _, r := range v.Results {
		if r.Name != "Google Feed Spec" {
			continue
		}
		if len(r.SubScores) == 3 {
			colWidth := v.Width / 3
			if colWidth < 20 {
				colWidth = 20
			}
			var cols []string
			for _, ss := range r.SubScores {
				bar := renderScoreBar(ss.Score, 14)
				col := lipgloss.NewStyle().Width(colWidth).Render(
					fmt.Sprintf("  %s\n  %s  %s",
						styles.StyleMetricLabel.Render(ss.Label),
						styles.StyleMetric.Render(fmt.Sprintf("%d/100", ss.Score)),
						bar,
					),
				)
				cols = append(cols, col)
			}
			fmt.Fprintf(&b, "\n%s\n", lipgloss.JoinHorizontal(lipgloss.Top, cols...))
		}
	}
	b.WriteString(v.renderCheckerTab("Google Feed Spec"))
	return b.String()
}

func (v ReportView) renderAITab() string {
	var b strings.Builder
	for _, r := range v.Results {
		if r.Name != "AI Readiness" {
			continue
		}
		fmt.Fprintf(&b, "\n  %s  %s\n\n", renderStatus(r.Status), r.Name)
		if r.Score != nil {
			scoreBar := renderScoreBar(*r.Score, 40)
			fmt.Fprintf(&b, "  Score: %s  %s\n\n", styles.StyleMetric.Render(fmt.Sprintf("%d/100", *r.Score)), scoreBar)
		}
		if len(r.Items) == 0 {
			fmt.Fprintf(&b, "  %s\n", styles.StyleCheckOK.Render("Fully optimized for AI."))
			return b.String()
		}
		fmt.Fprintf(&b, "  Improvements:\n\n")
		for _, item := range r.Items {
			bullet := styles.StyleMetricLabel.Render("  •")
			field := lipgloss.NewStyle().Bold(true).Render(item.Field)
			fmt.Fprintf(&b, "%s %s: %s\n", bullet, field, item.Message)
			for _, ex := range item.Examples {
				fmt.Fprintf(&b, "    %s\n", styles.StyleMetricLabel.Render(ex))
			}
		}
	}
	return b.String()
}

func renderStatus(s checker.Status) string {
	switch s {
	case checker.StatusOK:
		return styles.StyleStatusOK.Render("✓")
	case checker.StatusWarning:
		return styles.StyleStatusWarning.Render("⚠")
	case checker.StatusError:
		return styles.StyleStatusError.Render("✗")
	case checker.StatusFatal:
		return styles.StyleStatusFatal.Render("!")
	}
	return styles.StyleStatusFatal.Render("!")
}

func renderScoreBar(score, width int) string {
	filled := score * width / 100
	if filled > width {
		filled = width
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	var style lipgloss.Style
	switch {
	case score >= 80:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#10B981"))
	case score >= 50:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#F59E0B"))
	default:
		style = lipgloss.NewStyle().Foreground(lipgloss.Color("#EF4444"))
	}
	return style.Render(bar)
}

func (v ReportView) renderAttributesTab() string {
	if v.Feed == nil || len(v.Feed.Products) == 0 {
		return "  No products in feed.\n"
	}
	present := collectPresentAttrs(v.Feed)
	names := make([]string, 0, len(present))
	for name := range present {
		names = append(names, name)
	}
	sort.Strings(names)

	var b strings.Builder
	fmt.Fprintf(&b, "\n  %d attributes found in feed\n\n", len(names))
	for _, name := range names {
		meta, known := knownAttrs[name]
		badges := renderAttrBadges(meta, !known)
		fmt.Fprintf(&b, "  %-32s%s\n", name, badges)
	}
	return b.String()
}

func collectPresentAttrs(f *feed.Feed) map[string]struct{} {
	present := make(map[string]struct{})
	for i := range f.Products {
		p := &f.Products[i]
		checkField := func(name, val string) {
			if val != "" {
				present[name] = struct{}{}
			}
		}
		checkField("id", p.ID)
		checkField("title", p.Title)
		checkField("description", p.Description)
		checkField("price", p.Price)
		checkField("availability", p.Availability)
		checkField("link", p.Link)
		checkField("image_link", p.ImageLink)
		checkField("brand", p.Brand)
		checkField("gtin", p.GTIN)
		checkField("mpn", p.MPN)
		checkField("condition", p.Condition)
		checkField("color", p.Color)
		checkField("size", p.Size)
		checkField("material", p.Material)
		if len(p.AdditionalImages) > 0 {
			present["additional_image_link"] = struct{}{}
		}
		for k := range p.Extra {
			present[k] = struct{}{}
		}
	}
	return present
}

func renderAttrBadges(meta attrMeta, isCustom bool) string {
	var b strings.Builder
	if meta.GoogleRequired {
		b.WriteString(styles.StyleStatusError.Render("[Required]"))
	}
	if meta.GoogleRecommended {
		b.WriteString(styles.StyleStatusWarning.Render("[Recommended]"))
	}
	if meta.GoogleSupported {
		b.WriteString(styles.StyleMetric.Render("[Supported]"))
	}
	if meta.AIReady {
		b.WriteString(" " + styles.StyleStatusOK.Render("[AI]"))
	}
	if isCustom {
		b.WriteString(styles.StyleMetricLabel.Render("[Custom]"))
	}
	return b.String()
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

// BuildReport builds an exporter.Report from the feed and results.
func BuildReport(f *feed.Feed, results []checker.Result) exporter.Report {
	return exporter.Report{
		URL:          f.URL,
		FetchTime:    f.FetchTime,
		Size:         f.Size,
		ProductCount: f.ProductCount,
		Results:      results,
	}
}

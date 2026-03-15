package views

import (
	"fmt"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
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

// TabSupportsCursor returns true for tabs where items can be selected for drill-down.
func TabSupportsCursor(tab int) bool {
	switch tab {
	case TabXML, TabChecks, TabGoogleSpec, TabAI:
		return true
	}
	return false
}

// cursorDefault returns the initial cursor value for a tab (0 if it supports a cursor, -1 otherwise).
func cursorDefault(tab int) int {
	if TabSupportsCursor(tab) {
		return 0
	}
	return -1
}

// reportMode controls whether the report view is showing the normal tabbed view or drill-down.
type reportMode int

const (
	modeNormal    reportMode = iota
	modeDrillDown            // showing ProductListView for a selected item
)

// ExportSelections holds the format toggles chosen in the export overlay.
type ExportSelections struct {
	MainReport   bool
	ProductsCSV  bool
	ProductsJSON bool
}

// ReportView renders the tabbed report and export overlay.
type ReportView struct {
	Feed         *feed.Feed
	Results      []checker.Result
	ActiveTab    int
	SelectedItem int // cursor index into current tab's items; -1 on non-cursor tabs
	Viewport     viewport.Model
	ExportOpen   bool
	ExportInput  textinput.Model
	ExportMsg    string
	Width        int
	Height       int
	mode         reportMode
	drillDown    *ProductListView
	// export overlay toggle state
	exportMain      bool // default true
	exportCSV       bool // default true
	exportJSON      bool // default false
	exportFocusRow  int  // 0=filename, 1=main, 2=csv, 3=json
	exportToggleErr string
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
		Feed:         f,
		Results:      results,
		Viewport:     vp,
		ExportInput:  ti,
		Width:        width,
		Height:       height,
		SelectedItem: cursorDefault(TabOverview),
		exportMain:   true,
		exportCSV:    true,
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
	v.SelectedItem = cursorDefault(v.ActiveTab)
	v.Viewport.GotoTop()
	v.refreshViewport()
}

// PrevTab goes to the previous tab.
func (v *ReportView) PrevTab() {
	v.ActiveTab = (v.ActiveTab - 1 + TabCount) % TabCount
	v.SelectedItem = cursorDefault(v.ActiveTab)
	v.Viewport.GotoTop()
	v.refreshViewport()
}

// MoveCursorUp moves the item cursor up, if the current tab supports it.
func (v *ReportView) MoveCursorUp() {
	if !TabSupportsCursor(v.ActiveTab) {
		return
	}
	if v.SelectedItem > 0 {
		v.SelectedItem--
		v.refreshViewport()
	}
}

// MoveCursorDown moves the item cursor down, clamped to the number of items.
func (v *ReportView) MoveCursorDown() {
	if !TabSupportsCursor(v.ActiveTab) {
		return
	}
	items := v.currentItems()
	if v.SelectedItem < len(items)-1 {
		v.SelectedItem++
		v.refreshViewport()
	}
}

// CurrentSelectedItem returns a pointer to the currently selected item, or nil.
func (v *ReportView) CurrentSelectedItem() *checker.Item {
	if !TabSupportsCursor(v.ActiveTab) {
		return nil
	}
	items := v.currentItems()
	if len(items) == 0 || v.SelectedItem < 0 || v.SelectedItem >= len(items) {
		return nil
	}
	item := items[v.SelectedItem]
	return &item
}

// currentItems returns the checker.Items for the active tab.
func (v *ReportView) currentItems() []checker.Item {
	name := checkerNameForTab(v.ActiveTab)
	if name == "" {
		return nil
	}
	for _, r := range v.Results {
		if r.Name == name {
			if v.ActiveTab == TabAI {
				return sortedByImpact(r.Items)
			}
			return r.Items
		}
	}
	return nil
}

// checkerNameForTab maps a tab index to its checker result name.
func checkerNameForTab(tab int) string {
	switch tab {
	case TabXML:
		return "XML Validation"
	case TabChecks:
		return "Attribute Check"
	case TabGoogleSpec:
		return "Google Feed Spec"
	case TabAI:
		return "AI Readiness"
	}
	return ""
}

// OpenExport opens the export overlay, resetting toggles to defaults.
func (v *ReportView) OpenExport() {
	v.ExportOpen = true
	v.ExportInput.Focus()
	v.ExportMsg = ""
	v.exportMain = true
	v.exportCSV = true
	v.exportJSON = false
	v.exportFocusRow = 0
	v.exportToggleErr = ""
}

// ExportSelections returns the current toggle state of the export overlay.
func (v *ReportView) ExportSelections() ExportSelections {
	return ExportSelections{
		MainReport:   v.exportMain,
		ProductsCSV:  v.exportCSV,
		ProductsJSON: v.exportJSON,
	}
}

// HandleExportToggleKey processes navigation and toggle keys inside the export overlay.
// Returns true when the caller should confirm the export (Enter with valid selection).
func (v *ReportView) HandleExportToggleKey(key string) bool {
	const rows = 4 // 0=filename, 1=main, 2=csv, 3=json
	switch key {
	case "tab", "down":
		if v.exportFocusRow == 0 {
			v.ExportInput.Blur()
		}
		v.exportFocusRow = (v.exportFocusRow + 1) % rows
		if v.exportFocusRow == 0 {
			v.ExportInput.Focus()
		}
	case "shift+tab", "up":
		if v.exportFocusRow == 0 {
			v.ExportInput.Blur()
		}
		v.exportFocusRow = (v.exportFocusRow - 1 + rows) % rows
		if v.exportFocusRow == 0 {
			v.ExportInput.Focus()
		}
	case " ":
		switch v.exportFocusRow {
		case 1:
			v.exportMain = !v.exportMain
		case 2:
			v.exportCSV = !v.exportCSV
		case 3:
			v.exportJSON = !v.exportJSON
		}
		v.exportToggleErr = ""
	case "enter":
		if !v.exportMain && !v.exportCSV && !v.exportJSON {
			v.exportToggleErr = "Select at least one export format"
			return false
		}
		v.exportToggleErr = ""
		return true
	}
	return false
}

// InDrillDown reports whether the drill-down product list is active.
func (v *ReportView) InDrillDown() bool {
	return v.mode == modeDrillDown
}

// OpenDrillDown opens the product list drill-down for the currently selected item.
// Returns false if there is no selectable item with products.
func (v *ReportView) OpenDrillDown() bool {
	item := v.CurrentSelectedItem()
	if item == nil || len(item.AffectedProducts) == 0 {
		return false
	}
	checkerName := checkerNameForTab(v.ActiveTab)
	v.drillDown = NewProductListView(checkerName, item.Field, item.AffectedProducts, v.Width, v.Height)
	v.mode = modeDrillDown
	return true
}

// UpdateDrillDownMsg passes a tea.Msg to the drill-down view.
func (v *ReportView) UpdateDrillDownMsg(msg tea.Msg) (tea.Cmd, bool) {
	if v.drillDown == nil {
		return nil, true
	}
	updated, cmd, done := v.drillDown.Update(msg)
	v.drillDown = updated
	if done {
		v.mode = modeNormal
		v.drillDown = nil
	}
	return cmd, done
}

// CloseExport closes the export overlay.
func (v *ReportView) CloseExport() {
	v.ExportOpen = false
	v.ExportInput.Blur()
	v.ExportInput.SetValue("")
}

// View renders the full report view including any overlay.
func (v ReportView) View() string {
	if v.mode == modeDrillDown && v.drillDown != nil {
		return v.drillDown.View()
	}
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
	helpText := "  tab next  shift+tab prev  e export  q quit"
	if TabSupportsCursor(v.ActiveTab) {
		helpText = "  tab next  shift+tab prev  ↑/↓ select  enter drill-down  e export  q quit"
	}
	fmt.Fprintf(&b, "\n%s", styles.StyleHelp.Render(helpText))

	// Export success banner
	if v.ExportMsg != "" {
		fmt.Fprintf(&b, "  %s", styles.StyleSuccess.Render(v.ExportMsg))
	}

	return b.String()
}

func (v ReportView) renderWithOverlay(base string) string {
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

func (v ReportView) renderExportOverlay() string {
	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", styles.StyleTitle.Render("Export Report"))
	fmt.Fprintf(&b, "Filename: %s\n\n", v.ExportInput.View())

	// Toggle rows
	rows := []struct {
		label   string
		checked bool
		idx     int
	}{
		{"Main report  (.json / .md)", v.exportMain, 1},
		{"Product list (CSV)        ", v.exportCSV, 2},
		{"Product list (JSON)       ", v.exportJSON, 3},
	}
	for _, row := range rows {
		cursor := "  "
		if v.exportFocusRow == row.idx {
			cursor = styles.StyleMetric.Render("> ")
		}
		box := "[ ]"
		if row.checked {
			box = styles.StyleStatusOK.Render("[x]")
		}
		fmt.Fprintf(&b, "%s%s %s\n", cursor, box, row.label)
	}

	if v.exportToggleErr != "" {
		fmt.Fprintf(&b, "\n%s\n", styles.StyleStatusError.Render("  "+v.exportToggleErr))
	} else {
		fmt.Fprintln(&b)
	}
	fmt.Fprintf(&b, "%s", styles.StyleHelp.Render("enter export  ↑/↓ move  space toggle  esc cancel"))
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
		for i, item := range r.Items {
			cursor := "  "
			if TabSupportsCursor(v.ActiveTab) && i == v.SelectedItem {
				cursor = styles.StyleMetric.Render("▶ ")
			}
			bullet := styles.StyleMetricLabel.Render("•")
			field := lipgloss.NewStyle().Bold(true).Render(item.Field)
			fmt.Fprintf(&b, "%s%s %s: %s\n", cursor, bullet, field, item.Message)
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
		items := sortedByImpact(r.Items)
		for i, item := range items {
			cursor := "  "
			if i == v.SelectedItem {
				cursor = styles.StyleMetric.Render("▶ ")
			}
			bullet := styles.StyleMetricLabel.Render("•")
			field := lipgloss.NewStyle().Bold(true).Render(item.Field)
			fmt.Fprintf(&b, "%s%s %s: %s\n", cursor, bullet, field, item.Message)
			if item.Impact != "" {
				badge := renderImpactBadge(item.Impact)
				fmt.Fprintf(&b, "    %s  %s\n", badge, styles.StyleMetricLabel.Render(item.ImpactDesc))
			}
			if len(item.Examples) > 0 {
				fmt.Fprintln(&b)
				for _, ex := range item.Examples {
					fmt.Fprintf(&b, "      %s\n", styles.StyleMetricLabel.Render(ex))
				}
				fmt.Fprintln(&b)
			}
		}
	}
	return b.String()
}

func impactRank(impact string) int {
	switch impact {
	case "High":
		return 0
	case "Medium":
		return 1
	default:
		return 2
	}
}

func sortedByImpact(items []checker.Item) []checker.Item {
	out := make([]checker.Item, len(items))
	copy(out, items)
	sort.SliceStable(out, func(i, j int) bool {
		return impactRank(out[i].Impact) < impactRank(out[j].Impact)
	})
	return out
}

func renderImpactBadge(impact string) string {
	switch impact {
	case "High":
		return styles.StyleStatusOK.Render("[↑ High impact]")
	case "Medium":
		return styles.StyleStatusWarning.Render("[→ Medium impact]")
	default:
		return styles.StyleMetricLabel.Render("[↓ Low impact]")
	}
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
	groups := buildAttributeGroups(v.Feed)
	var b strings.Builder
	total := 0
	for _, g := range groups {
		total += len(g.Items)
	}
	fmt.Fprintf(&b, "\n  %d attributes found in feed\n\n", total)
	for _, g := range groups {
		if len(g.Items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "  %s\n", renderGroupDivider(g.Category, v.Width))
		for _, a := range g.Items {
			meta, known := knownAttrs[a.Name]
			badges := renderAttrBadges(meta, !known)
			cov := styles.StyleMetric.Render(fmt.Sprintf("%3d%%", a.Coverage))
			fmt.Fprintf(&b, "  %-32s%s  %s\n", a.Name, cov, badges)
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

func renderGroupDivider(label string, width int) string {
	prefix := "── " + label + " "
	rest := width - len(prefix) - 2
	if rest < 4 {
		rest = 4
	}
	return styles.StyleMetricLabel.Render(prefix + strings.Repeat("─", rest))
}

// buildAttributeGroups groups feed attributes by priority category with coverage percentages.
// It is the single source of truth consumed by both the TUI and BuildReport.
func buildAttributeGroups(f *feed.Feed) []exporter.AttributeGroup {
	coverage := collectAttrCoverage(f)
	total := len(f.Products)
	order := []string{"Required", "Recommended", "Supported", "Custom"}
	groups := make(map[string]*exporter.AttributeGroup, len(order))
	for _, cat := range order {
		groups[cat] = &exporter.AttributeGroup{Category: cat}
	}
	names := make([]string, 0, len(coverage))
	for name := range coverage {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		meta, known := knownAttrs[name]
		cat := primaryCategory(meta, !known)
		tags := attrTags(meta, !known)
		pct := 0
		if total > 0 {
			pct = coverage[name] * 100 / total
		}
		groups[cat].Items = append(groups[cat].Items, exporter.Attribute{Name: name, Tags: tags, Coverage: pct})
	}
	result := make([]exporter.AttributeGroup, 0, len(order))
	for _, cat := range order {
		result = append(result, *groups[cat])
	}
	return result
}

func primaryCategory(meta attrMeta, isCustom bool) string {
	switch {
	case isCustom:
		return "Custom"
	case meta.GoogleRequired:
		return "Required"
	case meta.GoogleRecommended:
		return "Recommended"
	case meta.GoogleSupported:
		return "Supported"
	default:
		return "Custom"
	}
}

func attrTags(meta attrMeta, isCustom bool) []string {
	var tags []string
	if meta.GoogleRequired {
		tags = append(tags, "Required")
	}
	if meta.GoogleRecommended {
		tags = append(tags, "Recommended")
	}
	if meta.GoogleSupported {
		tags = append(tags, "Supported")
	}
	if meta.AIReady {
		tags = append(tags, "AI")
	}
	if isCustom {
		tags = append(tags, "Custom")
	}
	return tags
}

// collectAttrCoverage returns a map of attribute name → count of products that have it.
func collectAttrCoverage(f *feed.Feed) map[string]int {
	counts := make(map[string]int)
	for i := range f.Products {
		p := &f.Products[i]
		countField := func(name, val string) {
			if val != "" {
				counts[name]++
			}
		}
		countField("id", p.ID)
		countField("title", p.Title)
		countField("description", p.Description)
		countField("price", p.Price)
		countField("availability", p.Availability)
		countField("link", p.Link)
		countField("image_link", p.ImageLink)
		countField("brand", p.Brand)
		countField("gtin", p.GTIN)
		countField("mpn", p.MPN)
		countField("condition", p.Condition)
		countField("color", p.Color)
		countField("size", p.Size)
		countField("material", p.Material)
		if len(p.AdditionalImages) > 0 {
			counts["additional_image_link"]++
		}
		for k := range p.Extra {
			counts[k]++
		}
	}
	return counts
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
		Attributes:   buildAttributeGroups(f),
	}
}

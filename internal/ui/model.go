// Package ui provides the Bubbletea TUI for merchant-analyzer.
package ui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/johlun99/merchant-analyzer/internal/analyzer"
	"github.com/johlun99/merchant-analyzer/internal/checker"
	"github.com/johlun99/merchant-analyzer/internal/exporter"
	"github.com/johlun99/merchant-analyzer/internal/feed"
	"github.com/johlun99/merchant-analyzer/internal/ui/views"
)

type viewState int

const (
	viewLoading viewState = iota
	viewReport
)

// CheckResultMsg carries a checker result back to the TUI.
type CheckResultMsg struct{ Result checker.Result }

// allDoneMsg signals that all checkers have finished.
type allDoneMsg struct{}

// exportDoneMsg signals that export completed.
type exportDoneMsg struct{ path string }

// exportErrMsg signals that export failed.
type exportErrMsg struct{ err error }

// clearExportMsgMsg clears the export status banner.
type clearExportMsgMsg struct{}

// Model is the root Bubbletea model.
type Model struct {
	feed     *feed.Feed
	checkers []checker.Checker
	cancel   context.CancelFunc
	program  *tea.Program

	state   viewState
	loading views.LoadingView
	report  *views.ReportView
	results []checker.Result
	mu      sync.Mutex

	width  int
	height int

	totalChecks int
	doneChecks  int
}

// New creates a new Model.
func New(f *feed.Feed, checkers []checker.Checker) *Model {
	names := make([]string, len(checkers))
	for i, c := range checkers {
		names[i] = c.Name()
	}
	return &Model{
		feed:        f,
		checkers:    checkers,
		loading:     views.NewLoadingView(f.URL, names),
		totalChecks: len(checkers),
	}
}

// SetProgram stores a reference to the tea.Program so we can send messages
// from background goroutines.
func (m *Model) SetProgram(p *tea.Program) {
	m.program = p
}

// Init starts the analysis and spinner.
func (m *Model) Init() tea.Cmd {
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel

	return tea.Batch(
		m.loading.Spinner.Tick,
		m.startAnalysis(ctx),
	)
}

func (m *Model) startAnalysis(ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		analyzer.Run(ctx, m.feed, m.checkers, func(r checker.Result) {
			if m.program != nil {
				m.program.Send(CheckResultMsg{Result: r})
			}
		})
		return allDoneMsg{}
	}
}

// Update handles incoming messages.
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKey(msg)

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.loading.Width = msg.Width
		if m.report != nil {
			m.report.SetSize(msg.Width, msg.Height)
		}

	case spinner.TickMsg:
		if m.state == viewLoading {
			var cmd tea.Cmd
			m.loading.Spinner, cmd = m.loading.Spinner.Update(msg)
			return m, cmd
		}

	case CheckResultMsg:
		m.mu.Lock()
		m.results = append(m.results, msg.Result)
		m.doneChecks++
		m.loading.MarkDone(msg.Result)
		m.mu.Unlock()

	case allDoneMsg:
		return m, m.transitionToReport()

	case exportDoneMsg:
		if m.report != nil {
			m.report.ExportMsg = fmt.Sprintf("✓ Exported to %s", msg.path)
			m.report.CloseExport()
		}
		return m, clearExportMsgAfter(2 * time.Second)

	case exportErrMsg:
		if m.report != nil {
			m.report.ExportMsg = fmt.Sprintf("✗ Export failed: %s", msg.err.Error())
			m.report.CloseExport()
		}

	case clearExportMsgMsg:
		if m.report != nil {
			m.report.ExportMsg = ""
		}
	}

	if m.state == viewReport && m.report != nil {
		return m.updateReport(msg)
	}

	return m, nil
}

func (m *Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Drill-down intercepts ALL keys (including q/ctrl+c) while active.
	if m.state == viewReport && m.report != nil && m.report.InDrillDown() {
		return m.handleDrillDownKey(msg)
	}

	if msg.String() == "ctrl+c" || msg.String() == "q" {
		if m.cancel != nil {
			m.cancel()
		}
		return m, tea.Quit
	}

	if m.state == viewReport && m.report != nil {
		return m.handleReportKey(msg)
	}

	return m, nil
}

func (m *Model) handleDrillDownKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	cmd, _ := m.report.UpdateDrillDownMsg(msg)
	if m.report.DrillDownWantsExport() {
		exportCmd := m.doDrillDownExport()
		m.report.CloseDrillDown()
		return m, exportCmd
	}
	return m, cmd
}

func (m *Model) doDrillDownExport() tea.Cmd {
	_, _, _, filename, products := m.report.DrillDownExportData()
	if filename == "" {
		filename = "products.csv"
	}
	return func() tea.Msg {
		var buf strings.Builder
		buf.WriteString("product_id,product_title,value\n")
		for _, p := range products {
			row := []string{p.ID, p.Title, p.Value}
			// Simple CSV encoding: quote fields containing comma, quote, or newline.
			encoded := make([]string, len(row))
			for i, f := range row {
				if strings.ContainsAny(f, ",\"\n") {
					f = `"` + strings.ReplaceAll(f, `"`, `""`) + `"`
				}
				encoded[i] = f
			}
			buf.WriteString(strings.Join(encoded, ",") + "\n")
		}
		if err := os.WriteFile(filename, []byte(buf.String()), 0o600); err != nil {
			return exportErrMsg{err: err}
		}
		return exportDoneMsg{path: filename}
	}
}

func (m *Model) handleReportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if m.report.ExportOpen {
		return m.handleExportKey(msg)
	}
	return m.handleNavigationKey(msg)
}

func (m *Model) handleExportKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if msg.String() == "esc" {
		m.report.CloseExport()
		return m, nil
	}
	if confirm := m.report.HandleExportToggleKey(msg.String()); confirm {
		return m, m.doExport(m.report.ExportInput.Value(), m.report.ExportSelections())
	}
	// Pass through to text input only when filename row is focused.
	return m.updateReport(msg)
}

func (m *Model) handleNavigationKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "tab":
		m.report.NextTab()
	case "shift+tab":
		m.report.PrevTab()
	case "e":
		m.report.OpenExport()
	case "enter":
		m.report.OpenDrillDown()
	case "up", "k":
		if views.TabSupportsCursor(m.report.ActiveTab) {
			m.report.MoveCursorUp()
		} else {
			m.report.Viewport.LineUp(1)
		}
	case "down", "j":
		if views.TabSupportsCursor(m.report.ActiveTab) {
			m.report.MoveCursorDown()
		} else {
			m.report.Viewport.LineDown(1)
		}
	}
	return m, nil
}

func (m *Model) updateReport(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.report.ExportOpen {
		var cmd tea.Cmd
		m.report.ExportInput, cmd = m.report.ExportInput.Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.report.Viewport, cmd = m.report.Viewport.Update(msg)
	return m, cmd
}

func (m *Model) transitionToReport() tea.Cmd {
	rv := views.NewReportView(m.feed, m.results, m.width, m.height)
	m.report = &rv
	m.state = viewReport
	return nil
}

func (m *Model) doExport(filename string, sel views.ExportSelections) tea.Cmd {
	filename = strings.TrimSpace(filename)
	if filename == "" {
		filename = "report.json"
	}
	report := views.BuildReport(m.feed, m.results)
	return func() tea.Msg {
		var written []string
		if sel.MainReport {
			if err := writeExport(report, filename); err != nil {
				return exportErrMsg{err: err}
			}
			written = append(written, filename)
		}
		base := strings.TrimSuffix(filename, filepath.Ext(filename))
		if sel.ProductsCSV {
			data, err := exporter.ToProductCSV(report)
			if err != nil {
				return exportErrMsg{err: err}
			}
			csvFile := base + "-products.csv"
			if err := os.WriteFile(csvFile, data, 0o600); err != nil {
				return exportErrMsg{err: err}
			}
			written = append(written, csvFile)
		}
		if sel.ProductsJSON {
			data, err := exporter.ToProductJSON(report)
			if err != nil {
				return exportErrMsg{err: err}
			}
			jsonFile := base + "-products.json"
			if err := os.WriteFile(jsonFile, data, 0o600); err != nil {
				return exportErrMsg{err: err}
			}
			written = append(written, jsonFile)
		}
		return exportDoneMsg{path: strings.Join(written, ", ")}
	}
}

// View renders the current state.
func (m *Model) View() string {
	switch m.state {
	case viewLoading:
		return m.loading.View()
	case viewReport:
		if m.report != nil {
			return m.report.View()
		}
		return m.loading.View()
	default:
		return m.loading.View()
	}
}

func clearExportMsgAfter(d time.Duration) tea.Cmd {
	return tea.Tick(d, func(_ time.Time) tea.Msg {
		return clearExportMsgMsg{}
	})
}

func writeExport(report exporter.Report, filename string) error {
	ext := strings.ToLower(filepath.Ext(filename))
	switch ext {
	case ".md":
		return os.WriteFile(filename, []byte(exporter.ToMarkdown(report)), 0o600)
	default:
		data, err := exporter.ToJSON(report)
		if err != nil {
			return err
		}
		return os.WriteFile(filename, data, 0o600)
	}
}

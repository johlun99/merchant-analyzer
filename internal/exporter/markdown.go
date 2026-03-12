package exporter

import (
	"fmt"
	"strings"

	"github.com/johlun99/merchant-analyzer/internal/checker"
)

// ToMarkdown renders a Report as a Markdown document.
func ToMarkdown(r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "# merchant-analyzer Report\n\n")
	fmt.Fprintf(&b, "## Summary\n\n")
	fmt.Fprintf(&b, "| Field | Value |\n|---|---|\n")
	fmt.Fprintf(&b, "| URL | %s |\n", r.URL)
	fmt.Fprintf(&b, "| Fetch time | %dms |\n", r.FetchTime.Milliseconds())
	fmt.Fprintf(&b, "| Size | %s |\n", formatBytes(r.Size))
	fmt.Fprintf(&b, "| Products | %d |\n", r.ProductCount)
	fmt.Fprintln(&b)

	fmt.Fprintf(&b, "## Results\n\n")
	for _, res := range r.Results {
		icon := statusIcon(res.Status)
		if res.Score != nil {
			fmt.Fprintf(&b, "### %s %s — Score: %d/100\n\n", icon, res.Name, *res.Score)
		} else {
			fmt.Fprintf(&b, "### %s %s\n\n", icon, res.Name)
		}

		if len(res.Items) == 0 {
			fmt.Fprintf(&b, "_No issues found._\n\n")
			continue
		}

		fmt.Fprintf(&b, "| Field | Message | Affected |\n|---|---|---|\n")
		for _, item := range res.Items {
			fmt.Fprintf(&b, "| `%s` | %s | %d |\n", item.Field, item.Message, item.Count)
		}
		fmt.Fprintln(&b)
	}

	b.WriteString(renderExamples(r.Results))
	b.WriteString(renderAttributes(r.Attributes))

	return b.String()
}

// renderAttributes returns a Markdown "## Attributes" section, or "" if no groups have items.
func renderAttributes(groups []AttributeGroup) string {
	hasItems := false
	for _, g := range groups {
		if len(g.Items) > 0 {
			hasItems = true
			break
		}
	}
	if !hasItems {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Attributes\n\n")
	for _, g := range groups {
		if len(g.Items) == 0 {
			continue
		}
		fmt.Fprintf(&b, "### %s (%d)\n\n", g.Category, len(g.Items))
		fmt.Fprintf(&b, "| Attribute | Coverage | Tags |\n|-----------|----------|------|\n")
		for _, a := range g.Items {
			fmt.Fprintf(&b, "| `%s` | %d%% | %s |\n", a.Name, a.Coverage, strings.Join(a.Tags, ", "))
		}
		fmt.Fprintln(&b)
	}
	return b.String()
}

// renderExamples returns a Markdown "## Examples" section, or "" if none exist.
func renderExamples(results []checker.Result) string {
	hasExamples := false
	for _, res := range results {
		for _, item := range res.Items {
			if len(item.Examples) > 0 {
				hasExamples = true
			}
		}
	}
	if !hasExamples {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "## Examples\n\n")
	for _, res := range results {
		sectionPrinted := false
		for _, item := range res.Items {
			if len(item.Examples) == 0 {
				continue
			}
			if !sectionPrinted {
				fmt.Fprintf(&b, "### %s\n\n", res.Name)
				sectionPrinted = true
			}
			fmt.Fprintf(&b, "**%s**\n\n", item.Field)
			for _, ex := range item.Examples {
				fmt.Fprintf(&b, "- %s\n", ex)
			}
			fmt.Fprintln(&b)
		}
	}
	return b.String()
}

func statusIcon(s checker.Status) string {
	switch s {
	case checker.StatusOK:
		return "✅"
	case checker.StatusWarning:
		return "⚠️"
	case checker.StatusError:
		return "❌"
	case checker.StatusFatal:
		return "💀"
	default:
		return "?"
	}
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

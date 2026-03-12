# merchant-analyzer

A CLI tool that analyzes merchant product feeds (XML/RSS) and reports on feed quality, broken attributes, and AI readiness.

## Features

- **XML Validation** — checks for well-formed, complete XML
- **Attribute Check** — validates required Google Shopping fields
- **Google Feed Spec** — weighted score across required, recommended, and format rules with per-product examples
- **AI Readiness** — scores feed against Google UCP, LLM attributes, and image quality; findings ordered by impact (High → Medium → Low)
- **Attributes** — all feed attributes grouped by priority (Required → Recommended → Supported → Custom) with coverage %
- **Metrics** — feed size, fetch time, product count
- **Interactive TUI** — live Bubbletea interface with tabbed report
- **Export** — JSON and Markdown reports including examples and attribute groups

## Usage

```bash
# Interactive TUI — remote URL
merchant-analyzer "https://example.com/feed.xml?locale=en"

# Interactive TUI — local file
merchant-analyzer /path/to/feed.xml

# Export report
merchant-analyzer "https://example.com/feed.xml" --output report.json
merchant-analyzer "https://example.com/feed.xml" --output report.md

# CI / non-interactive (exits with code 1 if errors found)
merchant-analyzer "https://example.com/feed.xml" --no-tui
```

> **Note:** Always quote URLs that contain query parameters (e.g. `?key=value&other=value`). Without quotes the shell interprets `&` as a background operator and splits the command.

## Installation

### Using `go install` (requires Go 1.21+)

```bash
# Latest version
go install github.com/johlun99/merchant-analyzer/cmd/merchant-analyzer@latest

# Specific version
go install github.com/johlun99/merchant-analyzer/cmd/merchant-analyzer@v1.2.0
```

The binary will be placed in `$(go env GOPATH)/bin`. Make sure that directory is in your `PATH`:

```bash
# Add to ~/.bashrc or ~/.zshrc if not already there
export PATH="$PATH:$(go env GOPATH)/bin"
```

### Build from source

```bash
git clone git@github.com:johlun99/merchant-analyzer.git
cd merchant-analyzer
go build -o merchant-analyzer ./cmd/merchant-analyzer
```

Move the binary to a directory in your `PATH`:

```bash
# macOS / Linux
sudo mv merchant-analyzer /usr/local/bin/
```

## Uninstall

### If installed via `go install`

```bash
rm "$(go env GOPATH)/bin/merchant-analyzer"
```

### If installed via binary / build from source

```bash
# macOS / Linux
which merchant-analyzer          # find the path
sudo rm /usr/local/bin/merchant-analyzer
```

### macOS — remove from PATH (if you added it)

Remove the `export PATH=...` line you added to `~/.bashrc`, `~/.zshrc`, or `~/.profile`, then reload:

```bash
source ~/.zshrc   # or ~/.bashrc
```

## License

MIT — see [LICENSE](LICENSE).

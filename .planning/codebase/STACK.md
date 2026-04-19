# Technology Stack

**Analysis Date:** 2026-04-19

## Languages

**Primary:**
- Go 1.26.1 - TUI application for LoRa device configuration

## Runtime

**Environment:**
- Go 1.26.1

**Package Manager:**
- Go Modules
- Lockfile: `go.sum` (present, managed)

## Frameworks

**Core:**
- Bubble Tea 1.3.10 - Terminal UI framework for building interactive CLI applications
- Bubbles 1.0.0 - Pre-built component library for Bubble Tea (text input, etc.)
- Lipgloss 1.1.0 - Style and layout library for terminal output

**Testing:**
- Testify 1.11.1 - Assertions and mocking for unit tests

**Serial Communication:**
- go.bug.st/serial 1.6.4 - Cross-platform serial port interface for communicating with Waveshare USB-TO-LoRa devices

## Key Dependencies

**Critical:**
- `go.bug.st/serial` 1.6.4 - Hardware interface for USB serial communication with LoRa devices; core to device connectivity
- `github.com/charmbracelet/bubbletea` 1.3.10 - TUI framework; implements the entire interactive UI layer
- `github.com/charmbracelet/bubbles` 1.0.0 - Provides text input components for parameter configuration
- `github.com/charmbracelet/lipgloss` 1.1.0 - Styling and layout for visual feedback (green/red borders on success/failure)

**Infrastructure:**
- `golang.org/x/sys` 0.38.0 - OS-level system calls for serial operations
- `golang.org/x/text` 0.3.8 - Text handling utilities

## Configuration

**Environment:**
- No environment variables required for core functionality
- Device path defaults to `/dev/ttyACM0` (configurable via TUI input)
- Serial baud rate defaults to 115200 (configurable via TUI)

**Build:**
- `Makefile` - Build automation with version tagging
- `.golangci.yml` - Go linting configuration (v2.11.4 of golangci-lint)
- `.pre-commit-config.yaml` - Pre-commit hooks for code quality checks

## Platform Requirements

**Development:**
- Go 1.26.1
- Pre-commit framework (Python 3.14+)
- golangci-lint v2.11.4
- POSIX-compatible shell for build scripts

**Production:**
- Linux, macOS (Intel/ARM), or Windows
- USB serial port access (device path varies by OS)
- No external services or cloud dependencies required

**Supported Binary Targets:**
- `linux/amd64`
- `darwin/amd64` (macOS Intel)
- `darwin/arm64` (macOS Apple Silicon)
- `windows/amd64`

## Build Configuration

**Makefile Targets:**
- `make build` - Compile binary with embedded version info (tag, commit, build date)
- `make clean` - Remove compiled binary
- `make test` - Run all Go tests

**Version Information (Embedded):**
- `main.Tag` - Git tag or "dev"
- `main.Commit` - Short commit hash
- `main.BuildDate` - Build timestamp (UTC)

**Cross-Platform Builds:**
- Controlled via `GOOS` and `GOARCH` environment variables
- Single Makefile rule produces platform-specific binaries with correct extensions

---

*Stack analysis: 2026-04-19*

# External Integrations

**Analysis Date:** 2026-04-19

## APIs & External Services

**None detected** - This is a standalone TUI application with no external APIs or cloud services.

## Data Storage

**Databases:**
- Not applicable - Application does not use databases

**File Storage:**
- Local filesystem only - No cloud storage or file service integrations

**Caching:**
- None - No caching layer required

## Hardware Integration

**Serial Communication:**
- Device: Waveshare USB-TO-LoRa-xF module
- Protocol: AT Command (ASCII text over serial port)
- Connection: USB serial port at configurable path (default: `/dev/ttyACM0`)
- Baud rate: Configurable (default: 115200)
- Implementation: `go.bug.st/serial` package
- Location: `internal/device/serial.go`, `internal/device/at.go`

**AT Command Protocol:**
- Three-phase session for each parameter change:
  1. Enter AT mode: Send `+++\r\n` (expect echo)
  2. Issue command: Send `AT+CMD=value\r\n` (expect `OK`)
  3. Exit AT mode: Send `AT+EXIT\r\n` (expect `OK`)
- Sessions are mutex-protected and sequential
- 1-second delay after `AT+EXIT` before next session starts
- All communication is text-based ASCII

**Supported AT Commands:**
- Configuration: `AT+SF`, `AT+BW`, `AT+CR`, `AT+PWR`, `AT+MODE`, `AT+LBT`, `AT+RSSI`, `AT+BAUD`, `AT+COMM`, `AT+NETID`, `AT+TXCH`, `AT+RXCH`, `AT+ADDR`, `AT+PORT`, `AT+KEY`
- Queries: `AT+ALLP?` (read all parameters), `AT+VER` (firmware version)
- Control: `AT+FACTORY` (restore defaults), `AT+RESET` (reboot device)
- Session: `+++` (enter mode), `AT+EXIT` (exit mode)

## Authentication & Identity

**Auth Provider:**
- None - No authentication required

**Device Access:**
- Direct USB serial port access
- No credentials or tokens needed
- User must have OS-level permissions to access serial device

## Monitoring & Observability

**Error Tracking:**
- None - No external error tracking service

**Logs:**
- Console stderr only - Application prints errors to stderr
- Location: `cmd/lora-config-sx1262/main.go` (lines 19-21)
- No persistent logging

## CI/CD & Deployment

**Hosting:**
- GitHub Releases (manual artifact hosting)
- Not deployed to cloud platform

**CI Pipeline:**
- GitHub Actions
- Build workflow: `.github/workflows/build.yml` - Triggered on PR to master with Go/Makefile/workflow changes
- Release workflow: `.github/workflows/release.yml` - Triggered on push to master with code changes
- Supported platforms: Linux amd64, macOS amd64, macOS arm64, Windows amd64

**Pre-Commit Hooks:**
- Local: `go-fmt`, `golangci-lint`, `go-build`, `go-mod-tidy`, `go-unit-tests`
- Standard: `trailing-whitespace`, `end-of-file-fixer`, `check-yaml`, `check-added-large-files` (10MB limit)

**Dependency Management:**
- Dependabot updates for `gomod` (weekly)
- Dependabot updates for GitHub Actions (weekly)
- Commit prefix: "deps" for gomod, "ci" for actions

## Environment Configuration

**Required env vars:**
- None - Application requires only device path and baud rate (both configurable via TUI at runtime)

**Secrets location:**
- No secrets used - Application is hardware-only communication tool

**Runtime Configuration:**
- Device path: Input via TUI (interactive prompt)
- Baud rate: Defaults to 115200, configurable via TUI
- All settings are temporary (not persisted) for current session

## Webhooks & Callbacks

**Incoming:**
- None

**Outgoing:**
- None

## Notes on Integration Simplicity

This application is intentionally self-contained with minimal external dependencies:

- **No network calls** - Operates entirely over USB serial
- **No external APIs** - Communicates only with local hardware
- **No authentication** - Direct hardware access only
- **No database** - State is in-memory during session
- **No third-party services** - Fully offline capable

The only external integration point is the USB serial device, which communicates via simple AT command protocol.

---

*Integration audit: 2026-04-19*

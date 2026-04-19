package device

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type SerialConn struct {
	port      Port
	sessionMu sync.Mutex // locks entire +++ → cmd → EXIT session
}

// NewSerialConn wraps an existing Port into a SerialConn.
func NewSerialConn(port Port) *SerialConn {
	return &SerialConn{port: port}
}

func OpenSerial(device string, baud int) (*SerialConn, error) {
	mode := &serial.Mode{
		BaudRate: baud,
		DataBits: 8,
		Parity:   serial.NoParity,
		StopBits: serial.OneStopBit,
	}
	port, err := serial.Open(device, mode)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", device, err)
	}
	if err := port.SetReadTimeout(100 * time.Millisecond); err != nil {
		port.Close()
		return nil, fmt.Errorf("set timeout: %w", err)
	}
	return &SerialConn{port: port}, nil
}

func (s *SerialConn) Close() error {
	s.sessionMu.Lock()
	defer s.sessionMu.Unlock()
	return s.port.Close()
}

// LockSession acquires the session mutex — caller must call UnlockSession when done.
func (s *SerialConn) LockSession() {
	s.sessionMu.Lock()
}

func (s *SerialConn) UnlockSession() {
	s.sessionMu.Unlock()
}

// sendAndRead writes a command and reads the response. NOT thread-safe — caller must hold session lock.
func (s *SerialConn) sendAndRead(cmd string) (string, error) {
	if err := s.port.ResetInputBuffer(); err != nil {
		return "", fmt.Errorf("reset buffer: %w", err)
	}

	_, err := s.port.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	var buf strings.Builder
	deadline := time.Now().Add(atTimeout)
	tmp := make([]byte, 256)

	for time.Now().Before(deadline) {
		bytesRead, err := s.port.Read(tmp)
		if bytesRead > 0 {
			buf.Write(tmp[:bytesRead])
			resp := buf.String()
			if strings.Contains(resp, "OK") || strings.Contains(resp, "ERROR") || strings.Contains(resp, "+++") {
				time.Sleep(interCmdDelay)
				return strings.TrimSpace(resp), nil
			}
		}
		if err != nil || bytesRead == 0 {
			time.Sleep(time.Millisecond) // yield to prevent CPU spin
			continue
		}
	}

	resp := buf.String()
	if resp == "" {
		return "", fmt.Errorf("timeout: no response")
	}
	return strings.TrimSpace(resp), fmt.Errorf("timeout: incomplete response: %s", resp)
}

// withATSession acquires the session lock, enters AT mode, runs callback, exits AT mode.
// If callback returns an error, exitAT is still attempted and both errors are reported.
// Callback error takes priority over exit error.
func (s *SerialConn) withATSession(callback func() error) error {
	s.LockSession()
	defer s.UnlockSession()

	if err := enterAT(s); err != nil {
		return err
	}

	callbackErr := callback()
	exitErr := exitAT(s)

	if callbackErr != nil {
		return callbackErr
	}
	return exitErr
}

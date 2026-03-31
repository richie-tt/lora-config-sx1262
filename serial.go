package main

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.bug.st/serial"
)

type SerialConn struct {
	port      serial.Port
	sessionMu sync.Mutex // locks entire +++ → cmd → EXIT session
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
func (s *SerialConn) sendAndRead(cmd string, timeout time.Duration) (string, error) {
	s.port.ResetInputBuffer()

	_, err := s.port.Write([]byte(cmd))
	if err != nil {
		return "", fmt.Errorf("write: %w", err)
	}

	var buf strings.Builder
	deadline := time.Now().Add(timeout)
	tmp := make([]byte, 256)

	for time.Now().Before(deadline) {
		n, err := s.port.Read(tmp)
		if n > 0 {
			buf.Write(tmp[:n])
			resp := buf.String()
			if strings.Contains(resp, "OK") || strings.Contains(resp, "ERROR") || strings.Contains(resp, "+++") {
				return strings.TrimSpace(resp), nil
			}
		}
		if err != nil {
			continue
		}
	}

	resp := buf.String()
	if resp == "" {
		return "", fmt.Errorf("timeout: no response")
	}
	return strings.TrimSpace(resp), nil
}

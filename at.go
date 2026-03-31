package main

import (
	"fmt"
	"strings"
	"time"
)

const atTimeout = 3 * time.Second
const postExitDelay = 1 * time.Second

// enterAT sends +++ and expects echo. Caller must hold session lock.
func enterAT(conn *SerialConn) error {
	resp, err := conn.sendAndRead("+++\r\n", atTimeout)
	if err != nil {
		return fmt.Errorf("enter AT mode: %w", err)
	}
	if !strings.Contains(resp, "+++") {
		return fmt.Errorf("enter AT mode: unexpected response: %s", resp)
	}
	return nil
}

// exitAT sends AT+EXIT and expects OK. Caller must hold session lock.
func exitAT(conn *SerialConn) error {
	resp, err := conn.sendAndRead("AT+EXIT\r\n", atTimeout)
	if err != nil {
		return fmt.Errorf("exit AT mode: %w", err)
	}
	if !strings.Contains(resp, "OK") {
		return fmt.Errorf("exit AT mode: unexpected response: %s", resp)
	}
	time.Sleep(postExitDelay)
	return nil
}

// SetParam: full atomic session +++ → AT+CMD=value → AT+EXIT
func SetParam(conn *SerialConn, atCmd, value string) error {
	conn.LockSession()
	defer conn.UnlockSession()

	if err := enterAT(conn); err != nil {
		return err
	}

	cmd := fmt.Sprintf("AT+%s=%s\r\n", atCmd, value)
	resp, err := conn.sendAndRead(cmd, atTimeout)
	if err != nil {
		exitAT(conn)
		return fmt.Errorf("set %s=%s: %w", atCmd, value, err)
	}
	if !strings.Contains(resp, "OK") {
		exitAT(conn)
		return fmt.Errorf("set %s=%s: %s", atCmd, value, resp)
	}

	return exitAT(conn)
}

// ReadAllParamsAndVersion: single session +++ → AT+ALLP? → AT+VER → AT+EXIT
func ReadAllParamsAndVersion(conn *SerialConn) (map[string]string, string, error) {
	conn.LockSession()
	defer conn.UnlockSession()

	if err := enterAT(conn); err != nil {
		return nil, "", err
	}

	resp, err := conn.sendAndRead("AT+ALLP?\r\n", atTimeout)
	if err != nil {
		exitAT(conn)
		return nil, "", fmt.Errorf("read ALLP: %w", err)
	}

	params, err := parseALLP(resp)
	if err != nil {
		exitAT(conn)
		return nil, "", err
	}

	// Read version in same session
	verResp, _ := conn.sendAndRead("AT+VER\r\n", atTimeout)
	version := parseVersion(verResp)

	if err := exitAT(conn); err != nil {
		return params, version, err
	}
	return params, version, nil
}

// Restore: full atomic session +++ → AT+RESTORE=1 → AT+EXIT
func Restore(conn *SerialConn) error {
	return SetParam(conn, "RESTORE", "1")
}

// Reboot: full atomic session +++ → AT+REBOOT (no EXIT — device reboots)
func Reboot(conn *SerialConn) error {
	conn.LockSession()
	defer conn.UnlockSession()

	if err := enterAT(conn); err != nil {
		return err
	}
	_, err := conn.sendAndRead("AT+REBOOT\r\n", atTimeout)
	return err
}

// parseALLP parses the +ALLP=... response.
// Order: SF,BW,CR,PWR,NETID,LBT,MODE,TXCH,RXCH,RSSI,ADDR,PORT,COMM,BAUD,KEY
func parseALLP(resp string) (map[string]string, error) {
	idx := strings.Index(resp, "+ALLP=")
	if idx < 0 {
		return nil, fmt.Errorf("parse ALLP: no +ALLP= in response: %s", resp)
	}

	data := resp[idx+6:]
	if nl := strings.IndexAny(data, "\r\n"); nl >= 0 {
		data = data[:nl]
	}

	parts := splitALLP(data)
	keys := []string{"SF", "BW", "CR", "PWR", "NETID", "LBT", "MODE", "TXCH", "RXCH", "RSSI", "ADDR", "PORT", "COMM", "BAUD", "KEY"}

	if len(parts) < len(keys) {
		return nil, fmt.Errorf("parse ALLP: expected %d fields, got %d: %s", len(keys), len(parts), data)
	}

	result := make(map[string]string, len(keys))
	for i, key := range keys {
		result[key] = parts[i]
	}
	return result, nil
}

func splitALLP(s string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false

	for _, r := range s {
		switch {
		case r == '"':
			inQuote = !inQuote
			current.WriteRune(r)
		case r == ',' && !inQuote:
			parts = append(parts, current.String())
			current.Reset()
		default:
			current.WriteRune(r)
		}
	}
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	return parts
}

func parseVersion(resp string) string {
	resp = strings.TrimSpace(resp)
	// Remove echoed command
	resp = strings.ReplaceAll(resp, "AT+VER", "")
	resp = strings.ReplaceAll(resp, "OK", "")
	// Handle +VER=... format
	if idx := strings.Index(resp, "+VER="); idx >= 0 {
		ver := resp[idx+5:]
		if nl := strings.IndexAny(ver, "\r\n"); nl >= 0 {
			ver = ver[:nl]
		}
		return strings.TrimSpace(ver)
	}
	return strings.TrimSpace(resp)
}

package device

import "time"

// Port defines the minimal serial port interface used by SerialConn.
// go.bug.st/serial.Port satisfies this interface implicitly.
type Port interface {
	Read(p []byte) (n int, err error)
	Write(p []byte) (n int, err error)
	Close() error
	ResetInputBuffer() error
	SetReadTimeout(t time.Duration) error
}

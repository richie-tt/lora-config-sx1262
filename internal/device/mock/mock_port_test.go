package mock

import (
	"errors"
	"testing"
	"time"

	tmock "github.com/stretchr/testify/mock"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPort_Read_CopiesData(t *testing.T) {
	port := new(Port)
	port.On("Read", tmock.Anything).Return([]byte("hello"), nil).Once()

	buf := make([]byte, 16)
	n, err := port.Read(buf)

	require.NoError(t, err)
	assert.Equal(t, 5, n)
	assert.Equal(t, "hello", string(buf[:n]))
	port.AssertExpectations(t)
}

func TestPort_Read_ReturnsError(t *testing.T) {
	port := new(Port)
	port.On("Read", tmock.Anything).Return(0, errors.New("read error")).Once()

	buf := make([]byte, 16)
	n, err := port.Read(buf)

	assert.Equal(t, 0, n)
	assert.EqualError(t, err, "read error")
}

func TestPort_Write(t *testing.T) {
	port := new(Port)
	port.On("Write", []byte("AT+CMD\r\n")).Return(8, nil).Once()

	n, err := port.Write([]byte("AT+CMD\r\n"))

	require.NoError(t, err)
	assert.Equal(t, 8, n)
	port.AssertExpectations(t)
}

func TestPort_Write_Error(t *testing.T) {
	port := new(Port)
	port.On("Write", tmock.Anything).Return(0, errors.New("write error")).Once()

	_, err := port.Write([]byte("data"))
	assert.EqualError(t, err, "write error")
}

func TestPort_Close(t *testing.T) {
	port := new(Port)
	port.On("Close").Return(nil).Once()

	require.NoError(t, port.Close())
	port.AssertExpectations(t)
}

func TestPort_Close_Error(t *testing.T) {
	port := new(Port)
	port.On("Close").Return(errors.New("close error")).Once()

	assert.EqualError(t, port.Close(), "close error")
}

func TestPort_ResetInputBuffer(t *testing.T) {
	port := new(Port)
	port.On("ResetInputBuffer").Return(nil).Once()

	require.NoError(t, port.ResetInputBuffer())
	port.AssertExpectations(t)
}

func TestPort_ResetInputBuffer_Error(t *testing.T) {
	port := new(Port)
	port.On("ResetInputBuffer").Return(errors.New("reset error")).Once()

	assert.EqualError(t, port.ResetInputBuffer(), "reset error")
}

func TestPort_SetReadTimeout(t *testing.T) {
	port := new(Port)
	port.On("SetReadTimeout", 100*time.Millisecond).Return(nil).Once()

	require.NoError(t, port.SetReadTimeout(100*time.Millisecond))
	port.AssertExpectations(t)
}

func TestPort_SetReadTimeout_Error(t *testing.T) {
	port := new(Port)
	port.On("SetReadTimeout", tmock.Anything).Return(errors.New("timeout error")).Once()

	assert.EqualError(t, port.SetReadTimeout(time.Second), "timeout error")
}

func TestForResponses(t *testing.T) {
	port := ForResponses("resp1", "resp2", "resp3")

	// Simulate 3 sendAndRead cycles
	for _, expected := range []string{"resp1", "resp2", "resp3"} {
		require.NoError(t, port.ResetInputBuffer())

		_, err := port.Write([]byte("cmd\r\n"))
		require.NoError(t, err)

		buf := make([]byte, 64)
		n, err := port.Read(buf)
		require.NoError(t, err)
		assert.Equal(t, expected, string(buf[:n]))
	}

	port.AssertExpectations(t)
}

func TestForResponses_Empty(t *testing.T) {
	port := ForResponses()

	require.NoError(t, port.ResetInputBuffer())
	_, err := port.Write([]byte("cmd\r\n"))
	require.NoError(t, err)

	port.AssertExpectations(t)
}

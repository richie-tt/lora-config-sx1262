package device

import (
	"errors"
	"testing"

	devicemock "lora-config-SX1262/internal/device/mock"

	"github.com/stretchr/testify/assert"
	tmock "github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

var errWrite = errors.New("write failed")

// --- enterAT / exitAT ---

func TestEnterAT_Success(t *testing.T) {
	port := devicemock.ForResponses("+++\r\nOK")
	conn := NewSerialConn(port)

	require.NoError(t, enterAT(conn))
	port.AssertExpectations(t)
}

func TestEnterAT_UnexpectedResponse(t *testing.T) {
	port := devicemock.ForResponses("ERROR")
	conn := NewSerialConn(port)

	assert.ErrorContains(t, enterAT(conn), "unexpected response")
}

func TestEnterAT_WriteError(t *testing.T) {
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", tmock.Anything).Return(0, errWrite)
	conn := NewSerialConn(port)

	assert.ErrorContains(t, enterAT(conn), "enter AT mode")
}

func TestExitAT_Success(t *testing.T) {
	port := devicemock.ForResponses("AT+EXIT\r\nOK")
	conn := NewSerialConn(port)

	require.NoError(t, exitAT(conn))
}

func TestExitAT_UnexpectedResponse(t *testing.T) {
	port := devicemock.ForResponses("ERROR")
	conn := NewSerialConn(port)

	assert.ErrorContains(t, exitAT(conn), "unexpected response")
}

// --- SetParam ---

func TestSetParam_Success(t *testing.T) {
	port := devicemock.ForResponses("+++", "OK", "OK")
	conn := NewSerialConn(port)

	require.NoError(t, SetParam(conn, "SF", "7"))
	port.AssertExpectations(t)
}

func TestSetParam_EnterATFails(t *testing.T) {
	port := devicemock.ForResponses("ERROR")
	conn := NewSerialConn(port)

	assert.ErrorContains(t, SetParam(conn, "SF", "7"), "enter AT mode")
}

func TestSetParam_CommandNotOK(t *testing.T) {
	port := devicemock.ForResponses("+++", "ERROR", "OK")
	conn := NewSerialConn(port)

	assert.ErrorContains(t, SetParam(conn, "SF", "99"), "set SF=99")
}

func TestSetParam_CommandWriteError(t *testing.T) {
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", []byte("+++\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("+++"), nil).Once()
	port.On("Write", []byte("AT+SF=7\r\n")).Return(0, errWrite)
	port.On("Write", []byte("AT+EXIT\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("OK"), nil).Once()
	conn := NewSerialConn(port)

	assert.Error(t, SetParam(conn, "SF", "7"))
}

// --- ReadAllParamsAndVersion ---

func TestReadAllParamsAndVersion_Success(t *testing.T) {
	allpResp := "+ALLP=7,0,1,22,0,0,1,10,10,0,0,0,\"8N1\",9600,0\r\nOK"
	port := devicemock.ForResponses("+++", allpResp, "+VER=1.2.3\r\nOK", "OK")
	conn := NewSerialConn(port)

	params, version, err := ReadAllParamsAndVersion(conn)
	require.NoError(t, err)
	assert.Equal(t, "7", params["SF"])
	assert.Equal(t, "22", params["PWR"])
	assert.Equal(t, `"8N1"`, params["COMM"])
	assert.Equal(t, "1.2.3", version)
}

func TestReadAllParamsAndVersion_EnterATFails(t *testing.T) {
	port := devicemock.ForResponses("ERROR")
	conn := NewSerialConn(port)

	params, version, err := ReadAllParamsAndVersion(conn)
	require.Error(t, err)
	assert.Nil(t, params)
	assert.Empty(t, version)
}

func TestReadAllParamsAndVersion_ALLPWriteError(t *testing.T) {
	port := new(devicemock.Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", []byte("+++\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("+++"), nil).Once()
	port.On("Write", []byte("AT+ALLP?\r\n")).Return(0, errWrite)
	port.On("Write", []byte("AT+EXIT\r\n")).Return(0, nil)
	port.On("Read", tmock.Anything).Return([]byte("OK"), nil).Once()
	conn := NewSerialConn(port)

	_, _, err := ReadAllParamsAndVersion(conn)
	assert.Error(t, err)
}

func TestReadAllParamsAndVersion_ALLPParseError(t *testing.T) {
	port := devicemock.ForResponses("+++", "OK", "OK")
	conn := NewSerialConn(port)

	_, _, err := ReadAllParamsAndVersion(conn)
	assert.ErrorContains(t, err, "parse ALLP")
}

// --- Restore / Reboot ---

func TestRestore_Success(t *testing.T) {
	port := devicemock.ForResponses("+++", "OK", "OK")
	conn := NewSerialConn(port)

	require.NoError(t, Restore(conn))
}

func TestReboot_Success(t *testing.T) {
	port := devicemock.ForResponses("+++", "OK")
	conn := NewSerialConn(port)

	require.NoError(t, Reboot(conn))
}

func TestReboot_EnterATFails(t *testing.T) {
	port := devicemock.ForResponses("ERROR")
	conn := NewSerialConn(port)

	assert.ErrorContains(t, Reboot(conn), "enter AT mode")
}

// --- Close ---

func TestClose(t *testing.T) {
	port := new(devicemock.Port)
	port.On("Close").Return(nil)
	conn := NewSerialConn(port)

	require.NoError(t, conn.Close())
	port.AssertExpectations(t)
}

func TestClose_Error(t *testing.T) {
	port := new(devicemock.Port)
	port.On("Close").Return(errors.New("close failed"))
	conn := NewSerialConn(port)

	assert.Error(t, conn.Close())
}

// --- NewSerialConn ---

func TestNewSerialConn(t *testing.T) {
	conn := NewSerialConn(new(devicemock.Port))
	require.NotNil(t, conn)
}

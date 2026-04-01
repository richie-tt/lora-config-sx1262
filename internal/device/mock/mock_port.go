package mock

import (
	"time"

	"github.com/stretchr/testify/mock"
)

// Port is a testify mock implementing device.Port.
type Port struct {
	mock.Mock
}

func (m *Port) Read(p []byte) (int, error) {
	args := m.Called(p)
	if data, ok := args.Get(0).([]byte); ok {
		copy(p, data)
		return len(data), args.Error(1)
	}
	return args.Int(0), args.Error(1)
}

func (m *Port) Write(p []byte) (int, error) {
	args := m.Called(p)
	return args.Int(0), args.Error(1)
}

func (m *Port) Close() error {
	return m.Called().Error(0)
}

func (m *Port) ResetInputBuffer() error {
	return m.Called().Error(0)
}

func (m *Port) SetReadTimeout(dur time.Duration) error {
	return m.Called(dur).Error(0)
}

// ForResponses creates a Port mock that answers a sequence of AT read responses.
// Each sendAndRead cycle triggers one ResetInputBuffer + Write + Read.
func ForResponses(responses ...string) *Port {
	port := new(Port)
	port.On("ResetInputBuffer").Return(nil)
	port.On("Write", mock.Anything).Return(0, nil)
	for _, resp := range responses {
		port.On("Read", mock.Anything).Return([]byte(resp), nil).Once()
	}
	return port
}

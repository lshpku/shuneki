package client

import (
	"io"
	"sync"
)

type sess struct {
	serial    uint16
	lowerConn io.ReadWriteCloser
	closed    bool
	mutex     sync.Mutex
}

func (s *sess) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.lowerConn.Write(p)
	s.mutex.Unlock()
	return n, err
}

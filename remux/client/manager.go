package client

import (
	"io"
	"sync"
	"time"
)

type Manager interface {
	// Submit submits a stream and lets Manager control it's data flow
	// and lifecycle
	Submit(stream io.ReadWriteCloser)
}

type manager struct {
	dial func() (io.ReadWriteCloser, error)

	// availConn may contain unavailble conns. Check everytime getting
	// a conn from it
	availConn *conn
	unavlConn *conn

	curConnCount  int
	curConnSerial int

	connMutex sync.Mutex

	// submitMutex ensures that dial() will not be called concurrently
	submitMutex sync.Mutex
}

func (m *manager) Submit(stream io.ReadWriteCloser) error {
	m.submitMutex.Lock()
	defer m.submitMutex.Unlock()

	// try using an existing conn
	m.connMutex.Lock()
	c := m.availConn
	for c != nil {
		c.mutex.Lock()

		// find the newest available conn
		if c.createTime.Add(time.Minute).After(time.Now()) &&
			c.curSessSerial < 16 {
			c.submit(stream)
		}

		// move unavailable conn to unavlConn
		cNext := c.next
		c.popSelf()
		c.pushFrontSelf(&m.unavlConn)
		c.mutex.Unlock()
		c = cNext
	}
	m.connMutex.Unlock()

	// open new conn
	c, err := newConn(m.dial)
	if err != nil {
		return err
	}
	c.submit(stream)

	m.connMutex.Lock()
	c.pushFrontSelf(&m.availConn)
	m.connMutex.Unlock()

	return nil
}

func NewManager(dial func() (io.ReadWriteCloser, error)) Manager {
	return nil
}

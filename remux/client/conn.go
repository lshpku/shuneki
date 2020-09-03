package client

import (
	"io"
	"sync"
	"time"
)

// conn corresponds to a lower connection
type conn struct {
	createTime time.Time
	lowerConn  io.ReadWriteCloser
	closed     bool

	// mutex is only for diaSess-related fields
	mutex         sync.Mutex
	curSessSerial uint16
	curSessCount  int
	sessMap       map[uint16]*sess

	prev *conn
	next *conn
}

func (c *conn) submit(stream io.ReadWriteCloser) {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	// add stream to receiver pool
	c.curSessCount++
	c.curSessSerial++
	c.sessMap[c.curSessSerial] = &sess{
		serial:    c.curSessSerial,
		lowerConn: stream,
		mutex:     sync.Mutex{},
	}

	// start a loop to forward stream
	go c.forwardSessLoop(c.curSessSerial)
}

func (c *conn) forwardSessLoop(serial uint16) {
	c.mutex.Lock()
	s, ok := c.sessMap[serial]
	if !ok {
		c.mutex.Unlock()
		return
	}
	c.mutex.Unlock()

	p := make([]byte, 65540)
	pOff := p[4:]
	for {
		n, err := s.lowerConn.Read(pOff)
		if err != nil {
			break
		}
		_, err = c.lowerConn.Write(p[:n+4])
		if err != nil {
			c.close()
			return
		}
	}

	_, err := c.lowerConn.Write(p[:4])
	if err != nil {
		c.close()
	}
}

// receiveLoop receives data in sections and writes to appropriate sess.
func (c *conn) receiveLoop() {
	p := make([]byte, 65540)
	var s *sess
	var size int
	var serial uint16

	for {
		readSize := size
		if readSize > len(p) {
			readSize = len(p)
		}
		n, err := c.lowerConn.Read(p[:size])
		if err != nil {
			return
		}

		// meet the end of sess
		if size == 0 {
			c.closeSess(serial)
			s = nil
			continue
		}

		// read the same section
		size -= n
		if s != nil {
			_, err = s.Write(p[:n])
		}

		// meet the end of section
		if size == 0 {
			s = nil
		}

		// read a new section
		c.mutex.Lock()
		s, ok := c.sessMap[uint16(p[0])]
		c.mutex.Unlock()
		if !ok {
			continue
		}
		s.Write(p[:n])
	}
}

// close closes lowerConn and all sess
func (c *conn) close() {
	c.mutex.Lock()
	if c.closed {
		c.mutex.Unlock()
		return
	}

	c.closed = true
	sessMap := c.sessMap
	c.sessMap = make(map[uint16]*sess)
	c.mutex.Unlock()

	for _, s := range sessMap {
		s.lowerConn.Close()
	}
	c.lowerConn.Close()
}

func (c *conn) closeSess(serial uint16) {
	// move sess out of the receivers
	c.mutex.Lock()
	s, ok := c.sessMap[serial]
	if !ok {
		c.mutex.Unlock()
		return
	}
	delete(c.sessMap, serial)
	c.mutex.Unlock()

	// close lowerConn of s
	s.lowerConn.Close()
}

func (c *conn) popSelf() {
	if c.prev != nil {
		c.prev.next = c.next
	}
	if c.next != nil {
		c.next.prev = c.prev
	}
}

func (c *conn) pushFrontSelf(head **conn) {
	c.next = *head
	c.prev = nil
	if *head != nil {
		(*head).prev = c
	}
	*head = c
}

func newConn(dial func() (io.ReadWriteCloser, error)) (*conn, error) {
	lowerConn, err := dial()
	if err != nil {
		return nil, err
	}
	return &conn{
		lowerConn:  lowerConn,
		mutex:      sync.Mutex{},
		createTime: time.Now(),
		sessMap:    make(map[uint16]*sess),
	}, nil
}

package eagersocks

import (
	"io"
)

type dumper struct {
	*baseStream
	request []byte
}

func (s *dumper) Read(p []byte) (int, error) {
	if len(s.request) == 0 {
		return s.stream.Read(p)
	}

	// read request as much as possible
	var nReq int
	if len(s.request) > len(p) {
		nReq = len(p)
	} else {
		nReq = len(s.request)
	}
	copy(p, s.request)
	p = p[nReq:]
	s.request = s.request[nReq:]

	// read content if there is space left
	if len(p) > 0 {
		n, err := s.stream.Read(p)
		return nReq + n, err
	}
	return nReq, nil
}

func (s *dumper) Write(p []byte) (int, error) {
	return s.stream.Write(p)
}

func (s *dumper) Close() error {
	return s.stream.Close()
}

func (s *dumper) Encoded() bool {
	return true
}

func (s *dumper) receiveHello() error {
	// meths(255)
	p := make([]byte, 255)

	// read version and nMeth
	_, err := io.ReadFull(s.stream, p[:2])
	if err != nil {
		return err
	}
	if p[0] != Version_SOCKS5 {
		return ErrInvalidVersion
	}
	nMeth := int(p[1])

	// read meths
	_, err = io.ReadFull(s.stream, p[:nMeth])
	if err != nil {
		return err
	}
	var supportsNoAuth bool
	for _, meth := range p[:nMeth] {
		if meth == Method_NoAuth {
			supportsNoAuth = true
			break
		}
	}
	if supportsNoAuth == false {
		return ErrInvalidMethod
	}

	return nil
}

func (s *dumper) replyHello() error {
	_, err := s.stream.Write([]byte{Version_SOCKS5, Method_NoAuth})
	return err
}

func (s *dumper) formatRequest() []byte {
	var p []byte

	// dump addr
	switch s.addressType {
	case AddressType_IPv4:
		p = make([]byte, 10)
		copy(p[4:8], s.address)
	case AddressType_Domain:
		p = make([]byte, 7+len(s.address))
		p[4] = uint8(len(s.address))
		copy(p[5:5+len(s.address)], s.address)
	case AddressType_IPv6:
		p = make([]byte, 22)
		copy(p[4:20], s.address)
	}

	// dump ver, cmd, rsv, port
	p[0] = Version_SOCKS5
	p[1] = s.command
	p[2] = 0x00
	p[3] = s.addressType
	p[len(p)-2] = uint8(s.port >> 8)
	p[len(p)-1] = uint8(s.port)

	return p
}

func (s *dumper) replyRequest() error {
	s.request = s.formatRequest()

	// use as rep here, but return to cmd afterward
	s.request[1] = 0x00
	defer func() {
		s.request[1] = s.command
	}()

	_, err := s.stream.Write(s.request)
	return err
}

// DumpStream dumps a socks-encoded stream to Stream, whose content is
// fsocks-encoded.
func DumpStream(stream io.ReadWriteCloser) (Stream, error) {
	s := &dumper{
		baseStream: &baseStream{
			stream: stream,
		},
	}

	err := s.receiveHello()
	if err != nil {
		return nil, err
	}
	err = s.replyHello()
	if err != nil {
		return nil, err
	}
	err = s.receiveRequest()
	if err != nil {
		return nil, err
	}
	err = s.replyRequest()
	if err != nil {
		return nil, err
	}

	return s, nil
}

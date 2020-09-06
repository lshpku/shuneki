package eagersocks

import (
	"errors"
	"io"
	"net"
	"strconv"
)

const (
	Version_SOCKS5 uint8 = 0x05

	Method_NoAuth uint8 = 0x00

	Command_CONNECT uint8 = 0x01
	Command_BIND    uint8 = 0x02
	Command_UDP     uint8 = 0x03

	AddressType_IPv4   uint8 = 0x01
	AddressType_Domain uint8 = 0x03
	AddressType_IPv6   uint8 = 0x04
)

var (
	ErrInvalidVersion     = errors.New("Invalid version")
	ErrInvalidMethod      = errors.New("Invalid method")
	ErrInvalidCommand     = errors.New("Invalid command")
	ErrInvalidReserve     = errors.New("Invalid reserve")
	ErrInvalidAddressType = errors.New("Invalid address type")
)

// Stream represents a stream with FSocks metadata
type Stream interface {
	// Close isn't recursive
	io.ReadWriteCloser

	Command() uint8
	AddressType() uint8
	Address() []byte
	Port() uint16
	Encoded() bool
	AddressString() string
}

type baseStream struct {
	stream      io.ReadWriteCloser
	command     uint8
	addressType uint8
	address     []byte
	port        uint16
}

func (s *baseStream) Command() uint8 {
	return s.command
}

func (s *baseStream) AddressType() uint8 {
	return s.addressType
}

func (s *baseStream) Address() []byte {
	return s.address
}

func (s *baseStream) Port() uint16 {
	return s.port
}

func (s *baseStream) AddressString() string {
	var addr string
	switch s.addressType {
	case AddressType_IPv4:
		addr = (net.IP)(s.address).String()
	case AddressType_Domain:
		addr = string(s.address)
	case AddressType_IPv6:
		addr = "[" + (net.IP)(s.address).String() + "]"
	default:
		return "<nil>"
	}
	return addr + ":" + strconv.Itoa(int(s.port))
}

// receiveRequest reads and parses a socks5 request
func (s *baseStream) receiveRequest() error {
	// addr(255) + port(2)
	p := make([]byte, 257)

	// read ver, cmd, rsv, atyp
	_, err := io.ReadFull(s.stream, p[:4])
	if err != nil {
		return err
	}

	// check version
	if p[0] != Version_SOCKS5 {
		return ErrInvalidVersion
	}

	// check command
	s.command = p[1]
	if s.command != Command_CONNECT { // TODO: support all commands
		return ErrInvalidCommand
	}

	// check reserve
	if p[2] != 0x00 {
		return ErrInvalidReserve
	}

	// check address type and read address and port
	s.addressType = p[3]
	var addrLen int
	switch s.addressType {
	case AddressType_IPv4:
		addrLen = 4
	case AddressType_Domain:
		_, err = io.ReadFull(s.stream, p[:1])
		if err != nil {
			return err
		}
		addrLen = int(p[0])
	case AddressType_IPv6:
		addrLen = 16
	default:
		return ErrInvalidAddressType
	}

	p = p[:addrLen+2]
	_, err = io.ReadFull(s.stream, p)
	if err != nil {
		return err
	}

	s.address = make([]byte, addrLen)
	copy(s.address, p)
	s.port = uint16(p[len(p)-2])<<8 | uint16(p[len(p)-1])
	return nil
}

package eagersocks

import "io"

type loader struct {
	*baseStream
}

func (s *loader) Read(p []byte) (int, error) {
	return s.stream.Read(p)
}

func (s *loader) Write(p []byte) (int, error) {
	return s.stream.Write(p)
}

func (s *loader) Close() error {
	return nil
}

func (s *loader) Encoded() bool {
	return false
}

// LoadStream loads an fsocks-encoded stream as Stream, whose content
// is not encoded.
func LoadStream(stream io.ReadWriteCloser) (Stream, error) {
	s := &loader{
		baseStream: &baseStream{
			stream: stream,
		},
	}

	err := s.receiveRequest()
	if err != nil {
		return nil, err
	}
	return s, nil
}

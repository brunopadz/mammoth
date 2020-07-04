package protocol

import "io"

func ReadMessageType(r io.Reader) (byte, error) {
	buf := []byte{0}
	_, err := r.Read(buf)
	return buf[0], err
}

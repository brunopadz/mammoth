package protocol

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
)

type Reader struct {
	r   *bufio.Reader
	Len int32
}

func ReadMessage(r io.Reader) (*Reader, error) {
	var sz int32
	err := binary.Read(r, binary.BigEndian, &sz)
	if err != nil {
		return nil, err
	}

	if sz < 4 {
		return nil, errors.New("Message size < 4 or overflow")
	}

	lr := io.LimitReader(r, int64(sz-4))
	return &Reader{
		Len: sz,
		r:   bufio.NewReader(lr),
	}, nil
}

func (r *Reader) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

func (r *Reader) ReadInt32() (i int32, err error) {
	err = binary.Read(r.r, binary.BigEndian, &i)
	return
}

func (r *Reader) ReadInt16() (i int16, err error) {
	err = binary.Read(r.r, binary.BigEndian, &i)
	return
}

func (r *Reader) ReadString() (string, error) {
	b, err := r.r.ReadBytes(0)
	if err != nil {
		return string(b), err
	}
	return string(b[0 : len(b)-1]), nil
}

func (r *Reader) ReadByte() (b byte, err error) {
	err = binary.Read(r.r, binary.BigEndian, &b)
	return
}

func (r *Reader) Finalize() error {
	n, err := r.r.WriteTo(ioutil.Discard)
	if err != nil {
		return err
	}
	if n != 0 {
		return errors.New("Message not fully consumed before finalization")
	}
	return nil
}

func (r *Reader) Discard() error {
	_, err := r.r.WriteTo(ioutil.Discard)
	return err
}

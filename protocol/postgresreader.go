package protocol

import (
	"bufio"
	"encoding/binary"
	"errors"
	"io"
	"io/ioutil"
)

func newReader(msgType byte, r io.Reader) (*PostgresReader, error) {
	var sz int32
	err := binary.Read(r, binary.BigEndian, &sz)
	if err != nil {
		return nil, err
	}

	if sz < 4 {
		return nil, errors.New("Message size < 4 or overflow")
	}

	lr := io.LimitReader(r, int64(sz-4))
	return NewPostgresReader(0x00, lr), nil
}

func ReadStartupMessage(r io.Reader) (*PostgresReader, error) {
	return newReader(0x00, r)
}

func ReadMessage(r io.Reader) (*PostgresReader, error) {
	var msgType byte
	err := binary.Read(r, binary.BigEndian, &msgType)
	if err != nil {
		return nil, err
	}

	return newReader(msgType, r)
}

type PostgresReader struct {
	MsgType byte
	r       *bufio.Reader
}

func NewPostgresReader(msgType byte, r io.Reader) *PostgresReader {
	return &PostgresReader{
		MsgType: msgType,
		r:       bufio.NewReader(r),
	}
}

func (r *PostgresReader) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

func (r *PostgresReader) ReadInt32() (i int32, err error) {
	err = binary.Read(r.r, binary.BigEndian, &i)
	return
}

func (r *PostgresReader) ReadInt16() (i int16, err error) {
	err = binary.Read(r.r, binary.BigEndian, &i)
	return
}

func (r *PostgresReader) ReadString() (string, error) {
	b, err := r.r.ReadBytes(0)
	if err != nil {
		return string(b), err
	}
	return string(b[0 : len(b)-1]), nil
}

func (r *PostgresReader) ReadByte() (b byte, err error) {
	err = binary.Read(r.r, binary.BigEndian, &b)
	return
}

func (r *PostgresReader) Finalize() error {
	n, err := r.Discard()
	if err != nil {
		return err
	}
	if n != 0 {
		return errors.New("Message not fully consumed before finalization")
	}
	return nil
}

func (r *PostgresReader) Discard() (int64, error) {
	return r.r.WriteTo(ioutil.Discard)
}

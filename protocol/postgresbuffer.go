package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

type PostgresBuffer struct {
	msgType     byte
	b           *bytes.Buffer
	multiReader io.Reader
}

func NewPostgresBuffer(msgType byte) *PostgresBuffer {
	return &PostgresBuffer{
		msgType: msgType,
		b:       &bytes.Buffer{},
	}
}

func (p *PostgresBuffer) Write(b []byte) error {
	// always returns nil error, according to doc'm
	p.b.Write(b)
	return nil
}

func (p *PostgresBuffer) WriteInt16(i int16) error {
	return binary.Write(p.b, binary.BigEndian, i)
}

func (p *PostgresBuffer) WriteInt32(i int32) error {
	return binary.Write(p.b, binary.BigEndian, i)
}

func (p *PostgresBuffer) WriteByte(b byte) error {
	return p.b.WriteByte(b)
}

func (p *PostgresBuffer) WriteString(str string) error {
	// err is always nil, according to doc'm
	p.b.WriteString(str)
	return p.b.WriteByte(0x00)
}

func (p *PostgresBuffer) WriteTo(w io.Writer) error {
	len := p.b.Len() + 4
	if len > math.MaxInt32 {
		return errors.New("Length of message too large")
	}

	if p.msgType != 0x00 {
		_, err := w.Write([]byte{p.msgType})
		if err != nil {
			return err
		}
	}

	err := binary.Write(w, binary.BigEndian, int32(len))
	if err != nil {
		return err
	}

	_, err = p.b.WriteTo(w)
	return err
}

func (p *PostgresBuffer) Read(b []byte) (int, error) {
	if p.multiReader == nil {
		var header *bytes.Buffer
		if p.msgType != 0 {
			header = bytes.NewBuffer(make([]byte, 5))
			header.WriteByte(p.msgType)
		} else {
			header = bytes.NewBuffer(make([]byte, 4))
		}
		err := binary.Write(header, binary.BigEndian, p.b.Len()+4)
		// should never fail, but...
		if err != nil {
			return 0, err
		}
		p.multiReader = io.MultiReader(header, p.b)
	}
	return p.multiReader.Read(b)
}

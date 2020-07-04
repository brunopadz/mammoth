package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"math"
)

type Buffer struct {
	msgType     byte
	b           *bytes.Buffer
	multiReader io.Reader
}

func NewPostgresBuffer(msgType byte) *Buffer {
	return &Buffer{
		msgType: msgType,
		b:       &bytes.Buffer{},
	}
}

func (p *Buffer) Write(b []byte) error {
	// always returns nil error, according to doc'm
	p.b.Write(b)
	return nil
}

func (p *Buffer) WriteInt16(i int16) error {
	return binary.Write(p.b, binary.BigEndian, i)
}

func (p *Buffer) WriteInt32(i int32) error {
	return binary.Write(p.b, binary.BigEndian, i)
}

func (p *Buffer) WriteByte(b byte) error {
	return p.b.WriteByte(b)
}

func (p *Buffer) WriteString(str string) error {
	// err is always nil, according to doc'm
	p.b.WriteString(str)
	return p.b.WriteByte(0x00)
}

func (p *Buffer) WriteTo(w io.Writer) error {
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

func (p *Buffer) Read(b []byte) (int, error) {
	len := p.b.Len() + 4
	if len > math.MaxInt32 {
		return 0, errors.New("Length of message too large")
	}

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

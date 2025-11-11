package langlang

import (
	"io"
	"unicode/utf8"
)

type MemInput struct {
	data []byte
	pos  int
}

func NewMemInput(data []byte) MemInput {
	return MemInput{data: data}
}

func (in *MemInput) PeekByte() (byte, error) {
	if in.pos >= len(in.data) {
		return 0, io.EOF
	}
	return in.data[in.pos], nil
}

func (in *MemInput) ReadByte() (byte, error) {
	b, err := in.PeekByte()
	if err != nil {
		return 0, err
	}
	in.pos++
	return b, nil
}

func (in *MemInput) PeekRune() (rune, int, error) {
	if in.pos >= len(in.data) {
		return 0, 0, io.EOF
	}
	if r := in.data[in.pos]; r < utf8.RuneSelf {
		return rune(r), 1, nil
	}
	r, size := utf8.DecodeRune(in.data[in.pos:])
	return r, size, nil
}

func (in *MemInput) ReadRune() (rune, int, error) {
	r, size, err := in.PeekRune()
	if err != nil {
		return 0, 0, err
	}
	in.pos += size
	return r, size, nil
}

func (in *MemInput) ReadString(start, end int) (string, error) {
	if start < 0 || end > len(in.data) {
		return "", io.EOF
	}
	return string(in.data[start:end]), nil
}

func (in *MemInput) Seek(n int) {
	in.pos = n
}

func (in *MemInput) Advance(n int) {
	in.pos += n
}

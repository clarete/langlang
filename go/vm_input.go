package langlang

import (
	"fmt"
	"io"
	"unicode/utf8"
)

type MemInput struct {
	data string
	pos  int
}

func NewMemInput(data string) *MemInput {
	return &MemInput{data: data}
}

func (in *MemInput) PeekRune() (rune, int, error) {
	if in.pos >= len(in.data) {
		return 0, 0, io.EOF
	}
	if r := in.data[in.pos]; r < utf8.RuneSelf {
		return rune(r), 1, nil
	}
	r, size := utf8.DecodeRuneInString(in.data[in.pos:])
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

func (in *MemInput) Seek(offset int64, whence int) error {
	if offset < 0 || int(offset) > len(in.data) {
		return fmt.Errorf("invalid seek offset")
	}
	if whence != io.SeekStart {
		return fmt.Errorf("invalid seek whence")
	}
	in.pos = int(offset)
	return nil
}

func (in *MemInput) ReadString(start, end int) (string, error) {
	if start < 0 || end > len(in.data) {
		return "", io.EOF
	}
	return string(in.data[start:end]), nil
}

func (in *MemInput) Advance(n int) {
	in.pos += n
}

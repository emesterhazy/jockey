package counter

import "io"

// Wraps an io.Reader and tracks the number of bytes read from it
type Reader struct {
	reader io.Reader
	count  int
}

// Read bytes into buf and return the number of bytes read and any error
func (r *Reader) Read(buf []byte) (n int, err error) {
	n, err = r.reader.Read(buf)
	r.count += n
	return
}

// Returns the number of bytes that have been read
func (r *Reader) Count() int {
	return r.count
}

// Returns a new reader that tracks the number of bytes read
func NewReader(r io.Reader) *Reader {
	return &Reader{reader: r}
}

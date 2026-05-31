package storage

import (
	"compress/gzip"
	"io"
)

// Compressor handles compression and decompression of data, and provides filename transformation for compressed
// files.
type Compressor interface {
	// Compress reads uncompressed data from [io.Reader] and writes compressed data to [io.Writer].
	Compress(r io.Reader, w io.Writer) error

	// Decompress reads compressed data from [io.Reader] and writes uncompressed data to [io.Writer].
	Decompress(r io.Reader, w io.Writer) error

	// NewReader returns a new [io.ReadCloser] that reads uncompressed data from r.
	NewReader(r io.Reader) (io.ReadCloser, error)

	// Filename returns the on-disk filename for a given base name, typically by appending a format-specific
	// extension.
	Filename(name string) string
}

// --------------------------------------------------------------------------------------------------------------------

// GzipCompressor implements [Compressor] using gzip compression.
type GzipCompressor struct {
	Level int // compression level, from gzip.NoCompression to gzip.BestCompression; 0 means gzip.NoCompression
	wpool *pool[*gzip.Writer]
	rpool *pool[*gzip.Reader]
}

var _ Compressor = GzipCompressor{} // compile-time interface assertion

// NewGzipCompressor returns a [GzipCompressor] with pool-backed writer and reader reuse.
func NewGzipCompressor(level int) GzipCompressor {
	if level < gzip.HuffmanOnly || level > gzip.BestCompression {
		level = gzip.NoCompression
	}

	return GzipCompressor{
		Level: level,
		wpool: newPool(func() *gzip.Writer {
			w, _ := gzip.NewWriterLevel(io.Discard, level) //nolint:errcheck // level already validated above

			return w
		}),
		rpool: newPool(func() *gzip.Reader { return new(gzip.Reader) }),
	}
}

// Compress implements [Compressor].
func (c GzipCompressor) Compress(r io.Reader, w io.Writer) error {
	gw := c.wpool.Get()
	gw.Reset(w)

	defer func() {
		gw.Reset(io.Discard)
		c.wpool.Put(gw)
	}()

	if _, err := io.Copy(gw, r); err != nil {
		_ = gw.Close()

		return err
	}

	return gw.Close()
}

// Decompress implements [Compressor].
func (c GzipCompressor) Decompress(r io.Reader, w io.Writer) error {
	gr := c.rpool.Get()
	defer c.rpool.Put(gr)

	if err := gr.Reset(r); err != nil {
		return err
	}

	if _, err := io.Copy(w, gr); err != nil {
		_ = gr.Close()

		return err
	}

	return gr.Close()
}

// NewReader implements [Compressor]. It returns a gzip reader that reads uncompressed data from r.
func (c GzipCompressor) NewReader(r io.Reader) (io.ReadCloser, error) {
	gr := c.rpool.Get()

	if err := gr.Reset(r); err != nil {
		c.rpool.Put(gr)

		return nil, err
	}

	return &pooledGzipReader{gr, c.rpool}, nil
}

// Filename implements [Compressor]. It appends ".gz" to name.
func (GzipCompressor) Filename(name string) string { return name + ".gz" }

type pooledGzipReader struct {
	*gzip.Reader

	pool *pool[*gzip.Reader]
}

var _ io.ReadCloser = (*pooledGzipReader)(nil) // ensure pooledGzipReader implements io.ReadCloser

func (r *pooledGzipReader) Close() error {
	if err := r.Reader.Close(); err != nil {
		return err
	}

	r.pool.Put(r.Reader)

	return nil
}

// --------------------------------------------------------------------------------------------------------------------

// NoopCompressor implements [Compressor] without any compression.
type NoopCompressor struct{}

var _ Compressor = NoopCompressor{} // compile-time interface assertion

// Compress implements [Compressor].
func (NoopCompressor) Compress(r io.Reader, w io.Writer) error { _, e := io.Copy(w, r); return e } //nolint:nlreturn

// Decompress implements [Compressor].
func (NoopCompressor) Decompress(r io.Reader, w io.Writer) error { _, e := io.Copy(w, r); return e } //nolint:nlreturn

// NewReader implements [Compressor].
func (NoopCompressor) NewReader(r io.Reader) (io.ReadCloser, error) { return io.NopCloser(r), nil }

// Filename implements [Compressor]. It returns name unchanged.
func (NoopCompressor) Filename(s string) string { return s }

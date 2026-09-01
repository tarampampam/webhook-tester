package storage_test

import (
	"bytes"
	"compress/gzip"
	"fmt"
	"io"
	"strings"
	"testing"

	"gh.tarampamp.am/webhook-tester/v3/internal/storage"
	"gh.tarampamp.am/webhook-tester/v3/internal/testutil/assert"
)

func TestGzipCompressor(t *testing.T) {
	t.Parallel()

	factory := func(*testing.T) storage.Compressor { return storage.NewGzipCompressor(gzip.BestSpeed) }

	RunCompressorSuite(t, factory)

	t.Run("invalid gzip data", func(t *testing.T) {
		t.Parallel()

		var out bytes.Buffer

		assert.Error(t, factory(t).Decompress(strings.NewReader("not gzip data"), &out))
	})

	t.Run("filename", func(t *testing.T) {
		t.Parallel()

		c := factory(t)

		assert.Equal(t, "session.json.gz", c.Filename("session.json"))
		assert.Equal(t, "foo.json.gz", c.Filename("foo.json"))
	})
}

func TestNoopCompressor(t *testing.T) {
	t.Parallel()

	factory := func(t *testing.T) storage.Compressor { return storage.NoopCompressor{} }

	RunCompressorSuite(t, factory)

	t.Run("filename", func(t *testing.T) {
		t.Parallel()

		c := factory(t)

		assert.Equal(t, "session.json", c.Filename("session.json"))
		assert.Equal(t, "foo.json", c.Filename("foo.json"))
	})
}

func RunCompressorSuite(t *testing.T, factory func(t *testing.T) storage.Compressor) {
	t.Helper()

	t.Run("round trip", func(t *testing.T) {
		t.Parallel()

		const input = "hello, webhook-tester! 😀"

		var compressed, decompressed bytes.Buffer

		c := factory(t)

		assert.NoError(t, c.Compress(strings.NewReader(input), &compressed))
		assert.NoError(t, c.Decompress(&compressed, &decompressed))
		assert.Equal(t, input, decompressed.String())
	})
}

// --------------------------------------------------------------------------------------------------------------------

func BenchmarkGzipCompressor(b *testing.B) {
	RunCompressorBenchmark(b, func(b *testing.B) storage.Compressor {
		return storage.NewGzipCompressor(gzip.DefaultCompression)
	})
}

func BenchmarkNoopCompressor(b *testing.B) {
	RunCompressorBenchmark(b, func(b *testing.B) storage.Compressor {
		return storage.NoopCompressor{}
	})
}

func RunCompressorBenchmark(b *testing.B, factory func(b *testing.B) storage.Compressor) {
	b.Helper()

	for _, payload := range [][]byte{
		[]byte(`{"method":"POST","url":"/hook","headers":{"Content-Type":"application/json"},"body":"hello"}`),
		[]byte(strings.Repeat(`{"id":1,"method":"POST","url":"/hook","body":"data"}`, 20)),
		[]byte(strings.Repeat(`{"id":1,"method":"POST","url":"/hook/path","headers":{"Content-Type":"application/json","X-Hub-Signature":"sha256=abc"},"body":"event-data"}`, 55)),
	} {
		size := fmt.Sprintf("%dB", len(payload))
		c := factory(b)

		var buf bytes.Buffer

		if err := c.Compress(bytes.NewReader(payload), &buf); err != nil {
			b.Fatalf("compress setup: %v", err)
		}

		compressed := buf.Bytes()

		b.Run(size+"/compress", func(b *testing.B) {
			b.ReportAllocs()

			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for range b.N {
				_ = c.Compress(bytes.NewReader(payload), io.Discard)
			}
		})

		b.Run(size+"/decompress", func(b *testing.B) {
			b.ReportAllocs()

			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for range b.N {
				_ = c.Decompress(bytes.NewReader(compressed), io.Discard)
			}
		})

		b.Run(size+"/new_reader", func(b *testing.B) {
			b.ReportAllocs()

			b.SetBytes(int64(len(payload)))
			b.ResetTimer()

			for range b.N {
				rc, err := c.NewReader(bytes.NewReader(compressed))
				if err != nil {
					b.Fatal(err)
				}

				_, _ = io.Copy(io.Discard, rc)
				_ = rc.Close()
			}
		})
	}
}

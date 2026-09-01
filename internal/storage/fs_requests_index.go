package storage

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"iter"
	"os"
	"path"
	"strconv"
	"time"
)

const (
	fsIndexRecordsSeparator byte = '\n' // appended after every encoded entry to delimit records
	fsIndexDelimiter        rune = ','
)

// fsRequestIndex manages the request index file (index.txt) inside a session's requests directory.
type fsRequestIndex struct {
	root  *os.Root
	dir   string // requests directory path within root, e.g. "{sHash}/requests"
	wPool *pool[*bufio.Writer]
}

// newFSRequestIndex creates a new fsRequestIndex backed by `root`. `dir` is the subdirectory within root where
// the index file lives; pass "." to operate directly in the root. Ownership of `root` is not transferred - the
// caller remains responsible for closing it.
func newFSRequestIndex(root *os.Root, dir string) *fsRequestIndex {
	return &fsRequestIndex{
		root:  root,
		dir:   dir,
		wPool: newPool(func() *bufio.Writer { return bufio.NewWriter(io.Discard) }),
	}
}

// indexPath returns the path to the index file within root.
func (idx *fsRequestIndex) indexPath() string { return path.Join(idx.dir, "index.txt") }

// atomicWrite passes a freshly created temp file to fn and, on success, closes and renames it over the index file.
// The rename makes the update atomic - readers see either the old file or the new one, never a partial write.
// On any error the temp file is removed and the index file is left untouched.
func (idx *fsRequestIndex) atomicWrite(fn func(io.Writer) error) error {
	tmpPath := idx.indexPath() + ".tmp"

	f, err := idx.root.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fsFilePerm)
	if err != nil {
		return err
	}

	defer func() { _ = idx.root.Remove(tmpPath) }() //nolint:errcheck

	bw := idx.wPool.Get()
	bw.Reset(f)

	defer func() {
		bw.Reset(io.Discard)
		idx.wPool.Put(bw)
	}()

	if err = fn(bw); err != nil {
		_ = f.Close()

		return err
	}

	if err = bw.Flush(); err != nil {
		_ = f.Close()

		return err
	}

	if err = f.Close(); err != nil {
		return err
	}

	return idx.root.Rename(tmpPath, idx.indexPath())
}

// ReadIndex opens the index file and returns a streaming iterator over its decoded entries.
// The returned error covers file-open failures, including [os.ErrNotExist].
//
// Decode errors during iteration are written to *errp (if non-nil) and stop the iterator.
//
// IMPORTANT: The caller must range over the returned iterator to release the file handle; an early break is safe.
func (idx *fsRequestIndex) ReadIndex(errp *error) (iter.Seq[fsRequestIndexRecord], error) {
	f, fErr := idx.root.Open(idx.indexPath())
	if fErr != nil {
		return nil, fErr
	}

	return func(yield func(fsRequestIndexRecord) bool) {
		defer func() {
			if err := f.Close(); err != nil && errp != nil && *errp == nil {
				*errp = err
			}
		}()

		sc := bufio.NewScanner(f)

		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}

			var r fsRequestIndexRecord

			if decErr := r.Decode(line); decErr != nil {
				if errp != nil {
					*errp = decErr
				}

				return
			}

			if !yield(r) {
				return
			}
		}

		if err := sc.Err(); err != nil && errp != nil {
			*errp = err
		}
	}, nil
}

// AddRecord encodes `r` and appends it to the index file (at the end), creating the file if absent.
//
// It makes no attempt to detect or prevent duplicate entries.
func (idx *fsRequestIndex) AddRecord(r fsRequestIndexRecord) (err error) {
	f, fErr := idx.root.OpenFile(idx.indexPath(), os.O_CREATE|os.O_WRONLY|os.O_APPEND, fsFilePerm)
	if fErr != nil {
		return fErr
	}

	defer func() {
		if cErr := f.Close(); cErr != nil && err == nil {
			err = cErr
		}
	}()

	if _, encErr := r.WriteTo(f); encErr != nil {
		return encErr
	}

	_, err = f.Write([]byte{fsIndexRecordsSeparator})

	return err
}

// DeleteRecords fully rewrites the index file, omitting every entry for which fn returns true.
//
// Returns [os.ErrNotExist] if the index file does not exist.
func (idx *fsRequestIndex) DeleteRecords(fn func(fsRequestIndexRecord) bool) error {
	src, err := idx.root.Open(idx.indexPath())
	if err != nil {
		return err
	}

	defer func() { _ = src.Close() }()

	return idx.atomicWrite(func(w io.Writer) error {
		br := bufio.NewReader(src)

		for {
			line, readErr := br.ReadBytes(fsIndexRecordsSeparator)

			if readErr != nil && !errors.Is(readErr, io.EOF) {
				return readErr
			}

			clearLine := line
			if len(clearLine) > 0 && clearLine[len(clearLine)-1] == fsIndexRecordsSeparator {
				clearLine = clearLine[:len(clearLine)-1]
			}

			if len(clearLine) > 0 {
				var r fsRequestIndexRecord

				if decErr := r.Decode(clearLine); decErr != nil {
					return decErr
				}

				if !fn(r) {
					if _, wErr := w.Write(line); wErr != nil {
						return wErr
					}
				}
			}

			if errors.Is(readErr, io.EOF) {
				return nil
			}
		}
	})
}

// WriteIndex atomically replaces the index file with the entries yielded by seq.
func (idx *fsRequestIndex) WriteIndex(seq iter.Seq[fsRequestIndexRecord]) error {
	return idx.atomicWrite(func(w io.Writer) error {
		for r := range seq {
			if _, encErr := r.WriteTo(w); encErr != nil {
				return encErr
			}

			if _, wErr := w.Write([]byte{fsIndexRecordsSeparator}); wErr != nil {
				return wErr
			}
		}

		return nil
	})
}

// --------------------------------------------------------------------------------------------------------------------

// fsRequestIndexRecord is a single entry in the per-session request index file.
type fsRequestIndexRecord struct { //nolint:recvcheck
	Hash      string
	CreatedAt time.Time
}

var _ io.WriterTo = (*fsRequestIndexRecord)(nil) // compile-time interface assertions

// WriteTo serializes the record as "hash,nanoseconds" and writes it to w.
func (r fsRequestIndexRecord) WriteTo(w io.Writer) (int64, error) {
	switch {
	case r.Hash == "":
		return 0, errors.New("empty request hash")
	case r.CreatedAt.IsZero():
		return 0, errors.New("missing CreatedAt timestamp")
	}

	var buf bytes.Buffer

	buf.Grow(len(r.Hash) + 1 + 20) //nolint:mnd // preallocate buffer for hash + comma + timestamp

	buf.WriteString(r.Hash)
	buf.WriteRune(fsIndexDelimiter)
	buf.WriteString(strconv.FormatInt(r.CreatedAt.UnixNano(), 10))

	return buf.WriteTo(w)
}

// Decode populates the record from a serialized line (separator already stripped).
func (r *fsRequestIndexRecord) Decode(data []byte) error {
	i := bytes.IndexRune(data, fsIndexDelimiter)
	if i < 0 {
		return errors.New("invalid index entry: not enough fields")
	}

	createdNs, pErr := strconv.ParseInt(string(data[i+1:]), 10, 64)
	if pErr != nil {
		return fmt.Errorf("invalid createdAt: %w", pErr)
	}

	r.Hash = string(data[:i])
	r.CreatedAt = time.Unix(0, createdNs).UTC()

	return nil
}

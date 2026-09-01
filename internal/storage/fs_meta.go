package storage

import (
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"time"
)

// fsSessionMeta is the metadata stored in the session metadata file in binary format (to speed up reads/writes).
type fsSessionMeta struct {
	CreatedAt time.Time // the time the session was created (zero value indicates corrupted/missing data)
	ExpiresAt time.Time // the time the session expires; zero means no expiration
	ID        string    // session ID
}

var ( // compile-time interface assertions
	_ io.WriterTo   = (*fsSessionMeta)(nil)
	_ io.ReaderFrom = (*fsSessionMeta)(nil)
)

// IsExpired returns true if the session metadata indicates that the session is expired as of the given time.
func (m *fsSessionMeta) IsExpired(now time.Time) bool {
	return !m.ExpiresAt.IsZero() && m.ExpiresAt.Before(now)
}

// toSessionMeta converts the internal session metadata to the public [SessionMeta] type.
func (m *fsSessionMeta) toSessionMeta() SessionMeta {
	return SessionMeta{CreatedAt: m.CreatedAt, ExpiresAt: m.ExpiresAt}
}

const (
	fsSmMagic string = "WHTSM/v1" // 8 bytes, used to identify and validate the session metadata file format

	fsSmMagicSize       = len(fsSmMagic) // 8 bytes                 [XXXXXXXX..................]
	fsSmCreatedAtNsSize = 8              // binary.Size(int64(0))   [........XXXXXXXX..........]
	fsSmExpiresAtNsSize = 8              // binary.Size(int64(0))   [................XXXXXXXX..]
	fsSmIDLenSize       = 2              // binary.Size(uint16(0))  [........................XX]

	fsSmHeaderSize = fsSmMagicSize + fsSmCreatedAtNsSize + fsSmExpiresAtNsSize + fsSmIDLenSize // 26 bytes
)

// ReadFrom reads the session metadata from r and populates m. It implements [io.ReaderFrom].
//
// The format is a fixed 26-byte header followed by a variable-length ID:
//
//	magic(8) | CreatedAtNs(8, BE) | ExpiresAtNs(8, BE) | idLen(2, BE) | ID(idLen)
//
// On any error the receiver is left in an undefined state.
func (m *fsSessionMeta) ReadFrom(r io.Reader) (int64, error) {
	var hdr [fsSmHeaderSize]byte // the fixed-size header fields are read into a single buffer for efficiency

	n, err := io.ReadFull(r, hdr[:]) // will fail if read less than len(hdr) bytes (io.ErrUnexpectedEOF)

	total := int64(n)
	if err != nil {
		return total, fmt.Errorf("read header: %w", err)
	}

	// magic
	off := 0
	if string(hdr[off:off+fsSmMagicSize]) != fsSmMagic {
		return total, fmt.Errorf("invalid magic: %q", hdr[off:off+fsSmMagicSize])
	}

	off += fsSmMagicSize

	// created at
	createdNs := int64(binary.BigEndian.Uint64(hdr[off : off+fsSmCreatedAtNsSize])) //nolint:gosec
	if createdNs == 0 {
		return total, errors.New("missing CreatedAt timestamp")
	}

	m.CreatedAt = time.Unix(0, createdNs).UTC()
	off += fsSmCreatedAtNsSize

	// expires at (0 ns = no expiration = zero time.Time)
	if expiresNs := int64(binary.BigEndian.Uint64(hdr[off : off+fsSmExpiresAtNsSize])); expiresNs != 0 { //nolint:gosec
		m.ExpiresAt = time.Unix(0, expiresNs).UTC()
	}

	off += fsSmExpiresAtNsSize

	// session id length
	idLen := binary.BigEndian.Uint16(hdr[off : off+fsSmIDLenSize])
	if idLen == 0 {
		return total, errors.New("empty session ID")
	}

	// session ID
	id := make([]byte, idLen)
	n, err = io.ReadFull(r, id)

	total += int64(n)
	if err != nil {
		return total, fmt.Errorf("read ID: %w", err)
	}

	m.ID = string(id)

	return total, nil
}

// WriteTo writes the session metadata to w. It implements [io.WriterTo].
//
// The format is a fixed 26-byte header followed by a variable-length ID:
//
//	magic(8) | CreatedAtNs(8, BE) | ExpiresAtNs(8, BE) | idLen(2, BE) | ID(idLen)
//
// Returns an error if CreatedAt is zero, ID is empty, or ID exceeds [math.MaxUint16] bytes.
func (m *fsSessionMeta) WriteTo(w io.Writer) (int64, error) {
	// validate up front - no partial writes on bad input
	switch {
	case m.CreatedAt.IsZero():
		return 0, errors.New("missing CreatedAt timestamp")
	case m.ID == "":
		return 0, errors.New("empty session ID")
	case len(m.ID) > math.MaxUint16:
		return 0, fmt.Errorf("session ID too long: %d bytes (max %d)", len(m.ID), math.MaxUint16)
	}

	// build the entire payload (header + ID) in a single buffer for one Write call
	buf := make([]byte, fsSmHeaderSize+len(m.ID))

	// magic
	off := 0
	copy(buf[off:off+fsSmMagicSize], fsSmMagic)
	off += fsSmMagicSize

	// created at
	binary.BigEndian.PutUint64(buf[off:off+fsSmCreatedAtNsSize], uint64(m.CreatedAt.UnixNano()))
	off += fsSmCreatedAtNsSize

	// expires at (zero time.Time = 0 ns on the wire = no expiration)
	var expiresNs int64
	if !m.ExpiresAt.IsZero() {
		expiresNs = m.ExpiresAt.UnixNano()
	}

	binary.BigEndian.PutUint64(buf[off:off+fsSmExpiresAtNsSize], uint64(expiresNs))
	off += fsSmExpiresAtNsSize

	// session id length
	binary.BigEndian.PutUint16(buf[off:off+fsSmIDLenSize], uint16(len(m.ID))) //nolint:gosec
	off += fsSmIDLenSize

	// session ID
	copy(buf[off:], m.ID)

	n, err := w.Write(buf)
	if err != nil {
		return int64(n), fmt.Errorf("write: %w", err)
	}

	return int64(n), nil
}

// --------------------------------------------------------------------------------------------------------------------

// fsRequestMeta is the metadata stored in the request metadata file in binary format (to speed up reads/writes).
type fsRequestMeta struct {
	CreatedAt time.Time // the time the request was created (zero value indicates corrupted/missing data)
	ID        string    // request ID
}

var ( // compile-time interface assertions
	_ io.WriterTo   = (*fsRequestMeta)(nil)
	_ io.ReaderFrom = (*fsRequestMeta)(nil)
)

const (
	fsRmMagic string = "WHTRM/v1" // 8 bytes, used to identify and validate the request metadata file format

	fsRmMagicSize       = len(fsRmMagic) // 8 bytes                 [XXXXXXXX..........]
	fsRmCreatedAtNsSize = 8              // binary.Size(int64(0))   [........XXXXXXXX..]
	fsRmIDLenSize       = 2              // binary.Size(uint16(0))  [................XX]

	fsRmHeaderSize = fsRmMagicSize + fsRmCreatedAtNsSize + fsRmIDLenSize // 18 bytes
)

// ReadFrom reads the request metadata from r and populates m. It implements [io.ReaderFrom].
//
// The format is a fixed 18-byte header followed by a variable-length ID:
//
//	magic(8) | CreatedAtNs(8, BE) | idLen(2, BE) | ID(idLen)
//
// On any error the receiver is left in an undefined state.
func (m *fsRequestMeta) ReadFrom(r io.Reader) (int64, error) {
	var hdr [fsRmHeaderSize]byte // the fixed-size header fields are read into a single buffer for efficiency

	n, err := io.ReadFull(r, hdr[:]) // will fail if read less than len(hdr) bytes (io.ErrUnexpectedEOF)

	total := int64(n)
	if err != nil {
		return total, fmt.Errorf("read header: %w", err)
	}

	// magic
	off := 0
	if string(hdr[off:off+fsRmMagicSize]) != fsRmMagic {
		return total, fmt.Errorf("invalid magic: %q", hdr[off:off+fsRmMagicSize])
	}

	off += fsRmMagicSize

	// created at
	createdNs := int64(binary.BigEndian.Uint64(hdr[off : off+fsRmCreatedAtNsSize])) //nolint:gosec
	if createdNs == 0 {
		return total, errors.New("missing CreatedAt timestamp")
	}

	m.CreatedAt = time.Unix(0, createdNs).UTC()
	off += fsRmCreatedAtNsSize

	// request id length
	idLen := binary.BigEndian.Uint16(hdr[off : off+fsRmIDLenSize])
	if idLen == 0 {
		return total, errors.New("empty request ID")
	}

	// request ID
	id := make([]byte, idLen)
	n, err = io.ReadFull(r, id)

	total += int64(n)
	if err != nil {
		return total, fmt.Errorf("read ID: %w", err)
	}

	m.ID = string(id)

	return total, nil
}

// toRequestMeta converts the internal request metadata to the public [RequestMeta] type.
func (m *fsRequestMeta) toRequestMeta() RequestMeta {
	return RequestMeta{CreatedAt: m.CreatedAt}
}

// WriteTo writes the request metadata to w. It implements [io.WriterTo].
//
// The format is a fixed 18-byte header followed by a variable-length ID:
//
//	magic(8) | CreatedAtNs(8, BE) | idLen(2, BE) | ID(idLen)
//
// Returns an error if CreatedAt is zero, ID is empty, or ID exceeds [math.MaxUint16] bytes.
func (m *fsRequestMeta) WriteTo(w io.Writer) (int64, error) {
	// validate up front - no partial writes on bad input
	switch {
	case m.CreatedAt.IsZero():
		return 0, errors.New("missing CreatedAt timestamp")
	case m.ID == "":
		return 0, errors.New("empty request ID")
	case len(m.ID) > math.MaxUint16:
		return 0, fmt.Errorf("request ID too long: %d bytes (max %d)", len(m.ID), math.MaxUint16)
	}

	// build the entire payload (header + ID) in a single buffer for one Write call
	buf := make([]byte, fsRmHeaderSize+len(m.ID))

	// magic
	off := 0
	copy(buf[off:off+fsRmMagicSize], fsRmMagic)
	off += fsRmMagicSize

	// created at
	binary.BigEndian.PutUint64(buf[off:off+fsRmCreatedAtNsSize], uint64(m.CreatedAt.UnixNano()))
	off += fsRmCreatedAtNsSize

	// request id length
	binary.BigEndian.PutUint16(buf[off:off+fsRmIDLenSize], uint16(len(m.ID))) //nolint:gosec
	off += fsRmIDLenSize

	// request ID
	copy(buf[off:], m.ID)

	n, err := w.Write(buf)
	if err != nil {
		return int64(n), fmt.Errorf("write: %w", err)
	}

	return int64(n), nil
}

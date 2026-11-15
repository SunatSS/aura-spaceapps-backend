package webauthn

import (
	"encoding/binary"
	"errors"
	"fmt"
)

// A minimal CBOR (RFC 8949) decoder for what WebAuthn sends: unsigned and
// negative integers, byte and text strings, arrays, maps, tags and simple
// values. Indefinite lengths are not used by authenticators and are rejected.

var errCBOR = errors.New("webauthn: malformed CBOR")

const maxItems = 1 << 12 // guards against absurd lengths

type decoder struct {
	b []byte
	i int
}

// decodeCBOR decodes one value from b and returns how many bytes it used.
func decodeCBOR(b []byte) (any, int, error) {
	d := &decoder{b: b}
	v, err := d.value(0)
	return v, d.i, err
}

func (d *decoder) arg(info byte) (uint64, error) {
	need := map[byte]int{24: 1, 25: 2, 26: 4, 27: 8}
	if info < 24 {
		return uint64(info), nil
	}
	n, ok := need[info]
	if !ok || d.i+n > len(d.b) {
		return 0, errCBOR
	}
	var v uint64
	switch n {
	case 1:
		v = uint64(d.b[d.i])
	case 2:
		v = uint64(binary.BigEndian.Uint16(d.b[d.i:]))
	case 4:
		v = uint64(binary.BigEndian.Uint32(d.b[d.i:]))
	case 8:
		v = binary.BigEndian.Uint64(d.b[d.i:])
	}
	d.i += n
	return v, nil
}

func (d *decoder) value(depth int) (any, error) {
	if depth > 16 || d.i >= len(d.b) {
		return nil, errCBOR
	}
	head := d.b[d.i]
	d.i++
	major, info := head>>5, head&0x1f
	if major == 7 {
		switch info {
		case 20:
			return false, nil
		case 21:
			return true, nil
		case 22, 23:
			return nil, nil
		case 25, 26, 27: // floats: skip, WebAuthn does not need them
			_, err := d.arg(info)
			return nil, err
		}
		return nil, errCBOR
	}
	n, err := d.arg(info)
	if err != nil {
		return nil, err
	}
	switch major {
	case 0:
		if n > 1<<62 {
			return nil, errCBOR
		}
		return int64(n), nil
	case 1:
		if n > 1<<62 {
			return nil, errCBOR
		}
		return -1 - int64(n), nil
	case 2, 3:
		if n > uint64(len(d.b)-d.i) {
			return nil, errCBOR
		}
		s := d.b[d.i : d.i+int(n)]
		d.i += int(n)
		if major == 3 {
			return string(s), nil
		}
		return append([]byte(nil), s...), nil
	case 4:
		if n > maxItems {
			return nil, errCBOR
		}
		arr := make([]any, 0, n)
		for range n {
			v, err := d.value(depth + 1)
			if err != nil {
				return nil, err
			}
			arr = append(arr, v)
		}
		return arr, nil
	case 5:
		if n > maxItems {
			return nil, errCBOR
		}
		m := make(map[any]any, n)
		for range n {
			k, err := d.value(depth + 1)
			if err != nil {
				return nil, err
			}
			switch k.(type) {
			case int64, string:
			default:
				return nil, fmt.Errorf("%w: unsupported map key %T", errCBOR, k)
			}
			v, err := d.value(depth + 1)
			if err != nil {
				return nil, err
			}
			m[k] = v
		}
		return m, nil
	case 6: // tag: return the tagged value
		return d.value(depth + 1)
	}
	return nil, errCBOR
}

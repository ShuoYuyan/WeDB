package main

/*
#include <stdint.h>
#include <stddef.h>
#include <string.h>

typedef int16_t  SQLSMALLINT;
typedef int32_t  SQLINTEGER;
typedef int64_t  SQLLEN;
typedef uint16_t SQLUSMALLINT;
typedef uint64_t SQLULEN;
typedef void*    SQLPOINTER;
typedef wchar_t  SQLWCHAR;

#define SQL_C_CHAR          1
#define SQL_C_WCHAR         (-8)
#define SQL_C_LONG          4
#define SQL_C_SLONG         (-16)
#define SQL_C_ULONG         (-18)
#define SQL_C_SHORT         5
#define SQL_C_SSHORT        (-15)
#define SQL_C_USHORT        (-17)
#define SQL_C_FLOAT         7
#define SQL_C_DOUBLE        8
#define SQL_C_BIT           (-7)
#define SQL_C_TINYINT       (-6)
#define SQL_C_STINYINT      (-26)
#define SQL_C_UTINYINT      (-28)
#define SQL_C_SBIGINT       (-25)
#define SQL_C_UBIGINT       (-27)
#define SQL_C_BINARY        (-2)
*/
import "C"

import (
	"fmt"
	"math"
	"strings"
	"unicode/utf8"
	"unsafe"
)

// goString converts a C string pointer to a Go string. If length > 0
// the bytes are read up to length; otherwise the C string is scanned
// for a NUL terminator.
func goString(p *C.char, length int) string {
	if p == nil {
		return ""
	}
	if length > 0 {
		b := C.GoBytes(unsafe.Pointer(p), C.int(length))
		return stripTrailingNul(string(b))
	}
	return C.GoString(p)
}

func stripTrailingNul(s string) string {
	if strings.HasSuffix(s, "\x00") {
		return s[:len(s)-1]
	}
	return s
}

// writeCString copies up to bufLen-1 bytes of s into dst, NUL-terminates,
// and returns the number of bytes the caller would need to hold the full
// string (truncation is silent, like SQLWCHAR behavior).
func writeCString(dst *C.char, s string, bufLen int) int {
	if dst == nil || bufLen <= 0 {
		return len(s)
	}
	max := bufLen - 1
	written := 0
	dstSlice := unsafe.Slice((*byte)(unsafe.Pointer(dst)), bufLen)
	for i := 0; i < len(s) && written < max; {
		r, size := utf8.DecodeRuneInString(s[i:])
		if r == utf8.RuneError && size == 1 {
			r = '?'
			size = 1
		}
		buf := make([]byte, 4)
		n := utf8.EncodeRune(buf, r)
		if written+n > max {
			break
		}
		copy(dstSlice[written:], buf[:n])
		written += n
		i += size
	}
	if written < bufLen {
		dstSlice[written] = 0
	}
	return len(s)
}

// mapGoToSQLType returns the ODBC SQL type code for a value whose
// internal WeDB type was set via inferColumnType. The mapping is
// identity for the small set of types we use.
func mapGoToSQLType(t int16) int16 {
	return t
}

// readSQLValue reads a parameter value from a caller buffer. Returns
// the string form so the SQL parser can splice it into a statement.
func readSQLValue(data *C.char, bufLen int64, outLen *C.SQLLEN, valueType int16) interface{} {
	if data == nil {
		if outLen != nil {
			*outLen = C.SQLLEN(-1) // SQL_NULL_DATA
		}
		return nil
	}
	length := bufLen
	if length <= 0 {
		length = 0
	}
	s := goString(data, int(length))
	for len(s) > 0 && s[len(s)-1] == 0 {
		s = s[:len(s)-1]
	}
	if outLen != nil && *outLen != C.SQLLEN(-1) {
		*outLen = C.SQLLEN(len(s))
	}
	return s
}

// writeSQLValue writes a value to a C buffer using SQL_C_CHAR
// conventions. Truncation is reported by setting *outLen to the full
// length.
func writeSQLValue(target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN, v interface{}) {
	writeSQLValueTyped(target, bufLen, outLen, v, C.SQL_C_CHAR)
}

// writeSQLValueTyped honors the caller's target type. For numeric
// target types (SQL_C_LONG/SQL_C_SHORT/SQL_C_SBIGINT/...) we copy the
// raw bytes of the value into the buffer; for character types we
// stringify.
func writeSQLValueTyped(target *C.char, bufLen C.SQLLEN, outLen *C.SQLLEN, v interface{}, targetType int16) {
	if target == nil {
		return
	}
	// bufLen==0 means "use the natural size of the target type".
	// Real ODBC clients pass the actual buffer size, but some
	// omit it for fixed-size numeric types.
	if bufLen == 0 {
		bufLen = C.SQLLEN(naturalSize(targetType))
	}
	if bufLen <= 0 {
		return
	}
	// SQL_C_NULL_DATA: caller wants to know value is NULL. We return
	// SQL_NULL_DATA via outLen and write nothing.
	if v == nil {
		if outLen != nil {
			*outLen = C.SQLLEN(-1) // SQL_NULL_DATA
		}
		return
	}
	var (
		s        string
		raw      []byte
		rawLen   int
		kind     = targetType
	)
	switch kind {
	case C.SQL_C_CHAR, C.SQL_C_WCHAR:
		s = formatSQLValue(v)
	case C.SQL_C_LONG, C.SQL_C_SLONG:
		raw, rawLen = encodeInt64(int64(toInt64(v)), 4)
	case C.SQL_C_SHORT, C.SQL_C_SSHORT:
		raw, rawLen = encodeInt64(int64(toInt64(v)), 2)
	case C.SQL_C_TINYINT, C.SQL_C_STINYINT:
		raw, rawLen = encodeInt64(int64(toInt64(v)), 1)
	case C.SQL_C_SBIGINT:
		raw, rawLen = encodeInt64(int64(toInt64(v)), 8)
	case C.SQL_C_UBIGINT, C.SQL_C_ULONG, C.SQL_C_USHORT, C.SQL_C_UTINYINT:
		raw, rawLen = encodeUint64(uint64(toInt64(v)), pickUSize(kind))
	case C.SQL_C_FLOAT, C.SQL_C_DOUBLE:
		raw, rawLen = encodeFloat64(toFloat64(v), pickFSize(kind))
	case C.SQL_C_BIT:
		if b, ok := v.(bool); ok && b {
			raw = []byte{1}
		} else {
			raw = []byte{0}
		}
		rawLen = 1
	default:
		// Unknown target type: stringify.
		s = formatSQLValue(v)
	}
	if outLen != nil {
		*outLen = C.SQLLEN(len(s))
	}
	if len(s) > 0 {
		writeCString(target, s, int(bufLen))
		return
	}
	dst := unsafe.Slice((*byte)(unsafe.Pointer(target)), int(bufLen))
	if rawLen > int(bufLen) {
		// Truncation: copy what fits.
		rawLen = int(bufLen)
	}
	copy(dst[:rawLen], raw[:rawLen])
	if outLen != nil {
		*outLen = C.SQLLEN(rawLen)
	}
}

func toInt64(v interface{}) int64 {
	switch x := v.(type) {
	case int:
		return int64(x)
	case int8:
		return int64(x)
	case int16:
		return int64(x)
	case int32:
		return int64(x)
	case int64:
		return x
	case uint:
		return int64(x)
	case uint8:
		return int64(x)
	case uint16:
		return int64(x)
	case uint32:
		return int64(x)
	case uint64:
		return int64(x)
	case float32:
		return int64(x)
	case float64:
		return int64(x)
	case bool:
		if x {
			return 1
		}
		return 0
	}
	return 0
}

func toFloat64(v interface{}) float64 {
	switch x := v.(type) {
	case int:
		return float64(x)
	case int32:
		return float64(x)
	case int64:
		return float64(x)
	case float32:
		return float64(x)
	case float64:
		return x
	}
	return 0
}

// naturalSize returns the byte size of a fixed-size target type.
// Returns 0 for variable-length types that require bufLen > 0.
func naturalSize(t int16) int {
	switch t {
	case C.SQL_C_LONG, C.SQL_C_SLONG, C.SQL_C_ULONG:
		return 4
	case C.SQL_C_SHORT, C.SQL_C_SSHORT, C.SQL_C_USHORT:
		return 2
	case C.SQL_C_TINYINT, C.SQL_C_STINYINT, C.SQL_C_UTINYINT, C.SQL_C_BIT:
		return 1
	case C.SQL_C_SBIGINT, C.SQL_C_UBIGINT:
		return 8
	case C.SQL_C_FLOAT:
		return 4
	case C.SQL_C_DOUBLE:
		return 8
	}
	return 0
}

func pickUSize(targetType int16) int {
	switch targetType {
	case C.SQL_C_UTINYINT:
		return 1
	case C.SQL_C_USHORT:
		return 2
	case C.SQL_C_ULONG:
		return 4
	case C.SQL_C_UBIGINT:
		return 8
	}
	return 8
}

func pickFSize(targetType int16) int {
	if targetType == C.SQL_C_FLOAT {
		return 4
	}
	return 8
}

func encodeInt64(v int64, size int) ([]byte, int) {
	out := make([]byte, size)
	switch size {
	case 1:
		out[0] = byte(int8(v))
	case 2:
		u := uint16(int16(v))
		out[0] = byte(u)
		out[1] = byte(u >> 8)
	case 4:
		u := uint32(int32(v))
		out[0] = byte(u)
		out[1] = byte(u >> 8)
		out[2] = byte(u >> 16)
		out[3] = byte(u >> 24)
	case 8:
		u := uint64(v)
		for i := 0; i < 8; i++ {
			out[i] = byte(u >> (i * 8))
		}
	default:
		return out, 0
	}
	return out, size
}

func encodeUint64(v uint64, size int) ([]byte, int) {
	out := make([]byte, size)
	switch size {
	case 1:
		out[0] = byte(v)
	case 2:
		u := uint16(v)
		out[0] = byte(u)
		out[1] = byte(u >> 8)
	case 4:
		u := uint32(v)
		out[0] = byte(u)
		out[1] = byte(u >> 8)
		out[2] = byte(u >> 16)
		out[3] = byte(u >> 24)
	case 8:
		for i := 0; i < 8; i++ {
			out[i] = byte(v >> (i * 8))
		}
	default:
		return out, 0
	}
	return out, size
}

func encodeFloat64(v float64, size int) ([]byte, int) {
	out := make([]byte, size)
	if size == 4 {
		u := uint32(int32(floatToFloat32Bits(v)))
		out[0] = byte(u)
		out[1] = byte(u >> 8)
		out[2] = byte(u >> 16)
		out[3] = byte(u >> 24)
		return out, 4
	}
	u := uint64(floatToFloat64Bits(v))
	for i := 0; i < 8; i++ {
		out[i] = byte(u >> (i * 8))
	}
	return out, 8
}

// We can't import math.Float32bits etc. without circular issues; use
// the standard library equivalents via a small wrapper.
func floatToFloat32Bits(f float64) uint32 { return math.Float32bits(float32(f)) }
func floatToFloat64Bits(f float64) uint64 { return math.Float64bits(f) }

func formatSQLValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch x := v.(type) {
	case string:
		return x
	case []byte:
		return string(x)
	case bool:
		if x {
			return "1"
		}
		return "0"
	}
	return fmt.Sprintf("%v", v)
}

// goStringFromW converts a SQLWCHAR* (UTF-16LE) into a Go string. If
// length is 0 or negative, the wide string is scanned for a NUL
// terminator up to a small bounded limit (256 wide chars). The
// bound prevents reading past the end of a caller-allocated
// buffer (the C test uses wconn[1024] on its stack; reading
// 8KB past the end of wconn would cross a Windows stack guard
// page and crash the process).
func goStringFromW(p *C.SQLWCHAR, length int) string {
	if p == nil {
		return ""
	}
	if length > 0 {
		b := unsafe.Slice((*uint16)(unsafe.Pointer(p)), length)
		for length > 0 && b[length-1] == 0 {
			length--
		}
		return string(utf16Decode(b[:length]))
	}
	const maxScan = 256
	b := unsafe.Slice((*uint16)(unsafe.Pointer(p)), maxScan)
	for i := 0; i < len(b); i++ {
		if b[i] == 0 {
			return string(utf16Decode(b[:i]))
		}
	}
	return string(utf16Decode(b[:]))
}

// writeWString copies a Go string into a SQLWCHAR* buffer encoded
// as UTF-16LE and NUL-terminated. The output buffer is assumed to
// hold at most bufLen wide chars. Characters that don't fit are
// truncated.
func writeWString(dst *C.SQLWCHAR, s string, bufLen int) {
	if dst == nil || bufLen <= 0 {
		return
	}
	b := unsafe.Slice((*uint16)(unsafe.Pointer(dst)), bufLen)
	max := bufLen - 1
	written := 0
	for _, r := range s {
		if written >= max {
			break
		}
		if r <= 0xFFFF {
			b[written] = uint16(r)
			written++
		} else {
			if written+1 >= max {
				break
			}
			r2 := r - 0x10000
			b[written] = 0xD800 | uint16(r2>>10)
			b[written+1] = 0xDC00 | uint16(r2&0x3F)
			written += 2
		}
	}
	if written < bufLen {
		b[written] = 0
	}
}

// utf16Decode decodes a UTF-16LE buffer to UTF-8 bytes. Surrogate
// pairs (BMP > 0xFFFF) are combined into a single rune.
func utf16Decode(b []uint16) []byte {
	out := make([]byte, 0, len(b)*2)
	for i := 0; i < len(b); {
		u := uint32(b[i])
		i++
		if u >= 0xD800 && u <= 0xDBFF && i < len(b) {
			lo := uint32(b[i])
			if lo >= 0xDC00 && lo <= 0xDFFF {
				u = 0x10000 + ((u - 0xD800) << 10) + (lo - 0xDC00)
				i++
			}
		}
		out = appendRuneUTF8(out, rune(u))
	}
	return out
}

func appendRuneUTF8(b []byte, r rune) []byte {
	switch {
	case r < 0x80:
		return append(b, byte(r))
	case r < 0x800:
		return append(b, 0xC0|byte(r>>6), 0x80|byte(r&0x3F))
	case r < 0x10000:
		return append(b, 0xE0|byte(r>>12), 0x80|byte((r>>6)&0x3F), 0x80|byte(r&0x3F))
	default:
		return append(b, 0xF0|byte(r>>18), 0x80|byte((r>>12)&0x3F), 0x80|byte((r>>6)&0x3F), 0x80|byte(r&0x3F))
	}
}

// cstr returns a *C.char pointing to a NUL-terminated copy of s.
// The buffer is allocated on the Go heap and is only valid for the
// synchronous call that follows (the W variants use this to forward
// to ANSI helpers in the same call frame).
func cstr(s string) *C.char {
	if s == "" {
		return nil
	}
	b := make([]byte, len(s)+1)
	copy(b, s)
	return (*C.char)(unsafe.Pointer(&b[0]))
}

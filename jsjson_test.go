package luxon

import (
	"fmt"
	"math"
	"strings"
)

// jsJSON is JSON.stringify for the values the tests produce; a top-level undefined prints as
// "undefined", the way the recorded rows show it.
func jsJSON(v any) string {
	if isUndef(v) {
		return "undefined"
	}
	var b strings.Builder
	writeJSON(&b, v)
	return b.String()
}

func writeJSON(b *strings.Builder, v any) {
	switch x := v.(type) {
	case nil, undefinedType:
		b.WriteString("null")
	case bool:
		fmt.Fprint(b, x)
	case float64:
		if math.IsNaN(x) || math.IsInf(x, 0) {
			b.WriteString("null")
		} else {
			b.WriteString(numberToString(x))
		}
	case string:
		writeJSONString(b, x)
	case *string:
		if x == nil {
			b.WriteString("null")
		} else {
			writeJSONString(b, *x)
		}
	case *bool:
		if x == nil {
			b.WriteString("null")
		} else {
			fmt.Fprint(b, *x)
		}
	case []any:
		b.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				b.WriteByte(',')
			}
			writeJSON(b, e)
		}
		b.WriteByte(']')
	case []*string:
		l := make([]any, len(x))
		for i, e := range x {
			l[i] = e
		}
		writeJSON(b, l)
	case []float64:
		l := make([]any, len(x))
		for i, e := range x {
			l[i] = e
		}
		writeJSON(b, l)
	case Obj:
		b.WriteByte('{')
		n := 0
		for i := 0; i+1 < len(x); i += 2 {
			if isUndef(x[i+1]) {
				continue
			}
			if n > 0 {
				b.WriteByte(',')
			}
			n++
			writeJSONString(b, x[i].(string))
			b.WriteByte(':')
			writeJSON(b, x[i+1])
		}
		b.WriteByte('}')
	default:
		if f, ok := asFloat(v); ok {
			writeJSON(b, f)
			return
		}
		fmt.Fprintf(b, "\"<%T>\"", v)
	}
}

func writeJSONString(b *strings.Builder, s string) {
	b.WriteByte('"')
	for _, r := range s {
		switch r {
		case '"':
			b.WriteString(`\"`)
		case '\\':
			b.WriteString(`\\`)
		case '\b':
			b.WriteString(`\b`)
		case '\f':
			b.WriteString(`\f`)
		case '\n':
			b.WriteString(`\n`)
		case '\r':
			b.WriteString(`\r`)
		case '\t':
			b.WriteString(`\t`)
		default:
			if r < 0x20 {
				fmt.Fprintf(b, `\u%04x`, r)
			} else {
				b.WriteRune(r)
			}
		}
	}
	b.WriteByte('"')
}

// jsCall runs fn the way the recording harness does: a JavaScript exception becomes
// {throw, msg}; a gap in the ICU data becomes {unsupported}.
func jsCall(fn func() any) (out string) {
	defer func() {
		if r := recover(); r != nil {
			switch e := r.(type) {
			case *Error:
				out = jsJSON(Obj{"throw", e.Name, "msg", e.Message})
			case *Unsupported:
				out = jsJSON(Obj{"unsupported", e.What})
			default:
				panic(r)
			}
		}
	}()
	return jsJSON(fn())
}

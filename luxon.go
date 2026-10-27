// Package luxon is a Go port of Luxon 3.7.2, the JavaScript date and time library, including
// the behaviour Luxon inherits from the ECMAScript Intl API (ICU and CLDR as compiled into
// Node.js 24), which internal/intl reproduces.
//
// Floating-point products that feed an addition are wrapped in float64(...) throughout: the
// conversion forbids the compiler from fusing them into one multiply-add, which JavaScript never
// does.
package luxon

// VERSION is Luxon's version.
const VERSION = "3.7.2"

// argAt is the i-th optional argument: undefined when it was not passed.
func argAt(args []any, i int) any {
	if i < len(args) {
		return args[i]
	}
	return Undefined
}

// optObj reads an optional trailing object argument as a fresh object: an Obj or internal
// object is copied, anything else (not passed, undefined or null) reads as {}.
func optObj(args []any, i int) *jsObj {
	switch o := argAt(args, i).(type) {
	case Obj:
		return objFrom(o)
	case *jsObj:
		if o != nil {
			return o.clone()
		}
	}
	return newObj()
}

// fnum reads a stored numeric value.
func fnum(v any) float64 {
	if f, ok := v.(float64); ok {
		return f
	}
	return toNumber(v)
}

// objOf builds an internal object from alternating keys and values.
func objOf(kv ...any) *jsObj { return objFrom(Obj(kv)) }

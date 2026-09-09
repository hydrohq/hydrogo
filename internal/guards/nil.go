// Package guards contains small boundary checks shared by infrastructure and
// domain adapters.
package guards

import "reflect"

// IsNil reports whether value is nil, including an interface containing a
// typed nil pointer, map, slice, function, channel, or interface.
func IsNil(value any) bool {
	if value == nil {
		return true
	}
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Ptr, reflect.Slice:
		return v.IsNil()
	default:
		return false
	}
}

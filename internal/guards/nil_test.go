package guards

import "testing"

func TestIsNilDetectsTypedNil(t *testing.T) {
	var pointer *int
	var function func()
	var value any = pointer
	if !IsNil(nil) || !IsNil(value) || !IsNil(function) {
		t.Fatal("typed nil dependencies were not detected")
	}
	if IsNil(struct{}{}) || IsNil(0) {
		t.Fatal("non-nil values were classified as nil")
	}
}

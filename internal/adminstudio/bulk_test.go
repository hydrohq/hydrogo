package adminstudio

import (
	"reflect"
	"testing"
)

func TestValidateBulkRequestNormalizesDeterministically(t *testing.T) {
	got, err := ValidateBulkRequest(BulkRequest{Action: " publish ", IDs: []string{" b ", "a"}})
	if err != nil {
		t.Fatal(err)
	}
	want := BulkRequest{Action: "publish", IDs: []string{"a", "b"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalized request = %#v, want %#v", got, want)
	}
}

func TestValidateBulkRequestRejectsUnsafeOrUnboundedInput(t *testing.T) {
	cases := []BulkRequest{
		{Action: "Publish", IDs: []string{"a"}},
		{Action: "publish/../../delete", IDs: []string{"a"}},
		{Action: "publish", IDs: []string{"a", "a"}},
		{Action: "publish", IDs: []string{"a\n"}},
		{Action: "publish", IDs: nil},
	}
	for _, request := range cases {
		if _, err := ValidateBulkRequest(request); err != ErrInvalidBulkRequest {
			t.Fatalf("request %#v error = %v", request, err)
		}
	}
}

func TestNormalizeBulkIDsRejectsLimitAboveContract(t *testing.T) {
	if _, err := NormalizeBulkIDs([]string{"a"}, DefaultBulkLimit+1); err != ErrInvalidBulkRequest {
		t.Fatalf("limit error = %v", err)
	}
}

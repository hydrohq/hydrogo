package adminstudio

import (
	"context"
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

func TestExecuteBulkIsDeterministicAndHidesCallbackErrors(t *testing.T) {
	var seen []string
	result, err := ExecuteBulk(context.Background(), BulkRequest{Action: "publish", IDs: []string{"b", "a", "c"}}, 10, func(_ context.Context, id string) error {
		seen = append(seen, id)
		if id == "b" {
			return context.DeadlineExceeded
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(seen, []string{"a", "b", "c"}) || result.Succeeded != 2 || result.Failed != 1 || !result.Completed {
		t.Fatalf("result = %#v seen = %#v", result, seen)
	}
	if result.Results[1].Code != "failed" || result.Results[1].Success {
		t.Fatalf("callback error leaked or success was marked: %#v", result.Results[1])
	}
}

func TestExecuteBulkStopsOnCancellationWithoutStartingNextItem(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := 0
	result, err := ExecuteBulk(ctx, BulkRequest{Action: "archive", IDs: []string{"a", "b"}}, 10, func(_ context.Context, id string) error {
		started++
		if id == "a" {
			cancel()
		}
		return nil
	})
	if err != context.Canceled || result.Completed || started != 1 || len(result.Results) != 1 {
		t.Fatalf("cancellation result = %#v started=%d err=%v", result, started, err)
	}
}

package transport

import (
	"sync/atomic"
	"testing"
)

// A panicking notifier must not crash the process: the goroutine outlives the
// handler, so net/http's own recovery does not cover it.
func TestGoNotificationRecoversPanic(t *testing.T) {
	var after atomic.Bool
	goNotification(nil, func() { panic("notifier adapter bug") })
	goNotification(nil, func() { after.Store(true) })
	WaitNotifications()
	if !after.Load() {
		t.Fatal("notification after a panic did not run")
	}
}

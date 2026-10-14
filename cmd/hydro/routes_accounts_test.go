package main

import "testing"

func TestPushRoutesSkipWhenSafeModeLeavesNoDispatcher(t *testing.T) {
	app := &serveApp{modules: map[string]bool{"push": true}}
	if routes := app.pushRoutes(); routes != nil {
		t.Fatalf("routes = %v, want none without a dispatcher", routes)
	}
}

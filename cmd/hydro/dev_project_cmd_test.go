package main

import (
	"bytes"
	"reflect"
	"sync"
	"testing"
)

func TestSplitDevArgs(t *testing.T) {
	services, rest, err := splitDevArgs([]string{"--address", "127.0.0.1:9", "--service", "ledger=go run ./automation", "--service=mail=./mailer -v"})
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 2 || services[0].name != "ledger" || !reflect.DeepEqual(services[0].argv, []string{"go", "run", "./automation"}) || services[1].name != "mail" || len(services[1].argv) != 2 {
		t.Fatalf("services = %+v", services)
	}
	if !reflect.DeepEqual(rest, []string{"--address", "127.0.0.1:9"}) {
		t.Fatalf("rest = %v", rest)
	}
	for _, bad := range [][]string{{"--service"}, {"--service", "noequals"}, {"--service", "x="}} {
		if _, _, err := splitDevArgs(bad); err == nil {
			t.Errorf("%v accepted", bad)
		}
	}
}

func TestPrefixWriterMarksWholeLines(t *testing.T) {
	var out bytes.Buffer
	w := &prefixWriter{mu: &sync.Mutex{}, out: &out, prefix: "[x]"}
	_, _ = w.Write([]byte("one\ntw"))
	_, _ = w.Write([]byte("o\n"))
	if out.String() != "[x] one\n[x] two\n" {
		t.Fatalf("got %q", out.String())
	}
}

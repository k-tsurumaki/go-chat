package trace

import (
	"testing"
	"bytes"
)

func TestNew(t *testing.T) {
	var buf bytes.Buffer
	tracer := New(&buf)
	if tracer == nil {
		t.Error("failed to create a Tracer")
	} else {
		tracer.Trace("hello trace package")
		if buf.String() != "hello trace package\n" {
			t.Errorf("got %q, want %q", buf.String(), "hello trace package\n")
		}
	}
}

func TestOff(t *testing.T) {
	var silentTracer Tracer = Off()
	silentTracer.Trace("hello trace package")
}
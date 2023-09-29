package nested

import (
	"fmt"
	"testing"
)

/*
// Original version:  Go 1.20+ required for type error to be comparable
func assertEqual[X comparable](t *testing.T, want, got X) {
	t.Helper()
	if want != got {
		t.Errorf("want %v, got %v", want, got)
	}
}
*/

// Go 1.19 workaround
func assertEqual[X any](t *testing.T, want, got X) {
	t.Helper()
	if w, g := fmt.Sprint(want), fmt.Sprint(got); w != g {
		t.Errorf("want %v, got %v", w, g)
	}
}

func TestName(t *testing.T) {
	assertEqual(t, "ready", Ready.String())
	assertEqual(t, "not ready", NotReady.String())
	assertEqual(t, "stopped", Stopped.String())
}

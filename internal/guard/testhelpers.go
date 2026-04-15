package guard

import "testing"

/*
SetModesForTest overrides SafeMode and DevMode for the duration of the test,
restoring prior values via t.Cleanup.

Intended for tests that exercise mode-sensitive behavior without touching the
process environment.

Since SafeMode/DevMode are package-level vars evaluated
once at init time, tests need a way to flip them deterministically.

Example:

	func TestSomething(t *testing.T) {
	    guard.SetModesForTest(t, false, true) // prod-mutations, dev-UI
	    // ... assertions ...
	}
*/
func SetModesForTest(t *testing.T, safe, dev bool) {
	t.Helper()
	origSafe, origDev := SafeMode, DevMode
	SafeMode, DevMode = safe, dev
	t.Cleanup(func() {
		SafeMode, DevMode = origSafe, origDev
	})
}

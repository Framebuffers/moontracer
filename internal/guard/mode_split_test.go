package guard_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"moontracer/internal/guard"
)

/*
Unit Testing: SAFE_MODE / DEV_MODE split.

	SafeMode gates Discord-mutating operations (roles, channels, threads, DMs).
	DevMode gates debug-only UI surfaces (Diagnostics, /campaigndatabase).
	They are independent; each of the four combinations represents a valid
	deployment posture.
*/

/*
Mode matrix covers every (SafeMode, DevMode) combination.

When:

	Each combo (t,t), (t,f), (f,t), (f,f) is applied via SetModesForTest.

Expected:

	Package vars reflect the applied values exactly. Each flag is orthogonal:
	setting one does not change the other. Each combo maps to a posture:
	  (t,t) → dev            (mutations faked, debug UI visible)
	  (t,f) → dev-no-debug   (mutations faked, debug UI hidden)
	  (f,t) → staging        (mutations live, debug UI visible)
	  (f,f) → production     (mutations live, debug UI hidden)
*/
func TestModeSplit_MatrixCoversAllPostures(t *testing.T) {
	cases := []struct {
		name string
		safe bool
		dev  bool
	}{
		{"dev (safe+dev)", true, true},
		{"dev-no-debug (safe only)", true, false},
		{"staging (dev only)", false, true},
		{"production (neither)", false, false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			guard.SetModesForTest(t, tc.safe, tc.dev)

			assert.Equal(t, tc.safe, guard.SafeMode, "SafeMode should match applied value")
			assert.Equal(t, tc.dev, guard.DevMode, "DevMode should match applied value")
		})
	}
}

/*
SetModesForTest restores original values on cleanup.

When:

	A subtest overrides SafeMode and DevMode via SetModesForTest, then returns.

Expected:

	After the subtest finishes, both flags are restored to their original
	values so subsequent tests are not polluted by ordering effects.
*/
func TestSetModesForTest_RestoresOriginalOnCleanup(t *testing.T) {
	origSafe, origDev := guard.SafeMode, guard.DevMode

	t.Run("override inside subtest", func(t *testing.T) {
		guard.SetModesForTest(t, !origSafe, !origDev)
		assert.NotEqual(t, origSafe, guard.SafeMode, "SafeMode should be overridden inside subtest")
		assert.NotEqual(t, origDev, guard.DevMode, "DevMode should be overridden inside subtest")
	})

	assert.Equal(t, origSafe, guard.SafeMode, "SafeMode should be restored after subtest cleanup")
	assert.Equal(t, origDev, guard.DevMode, "DevMode should be restored after subtest cleanup")
}

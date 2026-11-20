//go:build windowshost

package windowshost

import (
	"strings"
	"testing"
	"time"
)

// Probe: the same timing as before, with the zone changed by tzutil /s
// instead of Set-TimeZone, on a fresh runner.
func TestZZTzutilProbe(t *testing.T) {
	c := connect(t)
	orig := strings.TrimSpace(run(t, c, `(Get-TimeZone).Id`, nil).Text)
	step := func(label, script string, args map[string]any) {
		if script != "" {
			start := time.Now()
			run(t, c, script, args)
			t.Logf("%-26s set took %v", label, time.Since(start).Round(10*time.Millisecond))
		}
		var times []string
		for i := 0; i < 3; i++ {
			start := time.Now()
			run(t, c, `1`, nil)
			times = append(times, time.Since(start).Round(10*time.Millisecond).String())
		}
		zone := strings.TrimSpace(run(t, c, `(Get-TimeZone).Id`, nil).Text)
		t.Logf("%-26s calls %v | %s", label, times, zone)
	}
	tzutil := `param($A) & (Join-Path $env:WINDIR 'System32\tzutil.exe') /s $A.id; if ($LASTEXITCODE) { throw "tzutil exited $LASTEXITCODE" }`
	step("start ("+orig+")", "", nil)
	step("tzutil Tokyo", tzutil, map[string]any{"id": "Tokyo Standard Time"})
	step("tzutil back to "+orig, tzutil, map[string]any{"id": orig})
	step("tzutil Cape Verde", tzutil, map[string]any{"id": "Cape Verde Standard Time"})
	step("tzutil back again", tzutil, map[string]any{"id": orig})
	step("Set-TimeZone Tokyo", `param($A) Set-TimeZone -Id $A.id`, map[string]any{"id": "Tokyo Standard Time"})
	step("Set-TimeZone back", `param($A) Set-TimeZone -Id $A.id`, map[string]any{"id": orig})
}

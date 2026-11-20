//go:build windowshost

package windowshost

import (
	"strings"
	"testing"
	"time"
)

// Probe: how long a PowerShell call through Conn3x takes before and after
// Set-TimeZone, with the zone ID in Windows' own case and in lower case, and
// what Windows stores each time.
func TestZZTimezoneProbe(t *testing.T) {
	c := connect(t)
	orig := strings.TrimSpace(run(t, c, `(Get-TimeZone).Id`, nil).Text)
	step := func(label, script string, args map[string]any) {
		if script != "" {
			start := time.Now()
			run(t, c, script, args)
			t.Logf("%-28s set took %v", label, time.Since(start).Round(10*time.Millisecond))
		}
		var times []string
		for i := 0; i < 3; i++ {
			start := time.Now()
			run(t, c, `1`, nil)
			times = append(times, time.Since(start).Round(10*time.Millisecond).String())
		}
		state := strings.TrimSpace(run(t, c, `$k = Get-ItemProperty -LiteralPath 'HKLM:\SYSTEM\CurrentControlSet\Control\TimeZoneInformation'; "$((Get-TimeZone).Id) | key=$($k.TimeZoneKeyName) | dyn=$($k.DynamicDaylightTimeDisabled)"`, nil).Text)
		t.Logf("%-28s calls %v | %s", label, times, state)
	}
	set := `param($A) Set-TimeZone -Id $A.id`
	step("start ("+orig+")", "", nil)
	step("Tokyo, Windows' case", set, map[string]any{"id": "Tokyo Standard Time"})
	step("back to "+orig, set, map[string]any{"id": orig})
	step("tokyo, lower case", set, map[string]any{"id": "tokyo standard time"})
	step("back to "+orig+" again", set, map[string]any{"id": orig})
	step("one minute later", `Start-Sleep -Seconds 60`, nil)
	start := time.Now()
	out := strings.TrimSpace(run(t, c, `$s = [Diagnostics.Stopwatch]::StartNew(); $null = & powershell.exe -NoProfile -NonInteractive -Command 1; "inner powershell.exe took $($s.ElapsedMilliseconds) ms"`, nil).Text)
	t.Logf("%s (whole call %v)", out, time.Since(start).Round(10*time.Millisecond))
}

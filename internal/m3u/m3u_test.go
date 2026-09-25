package m3u

import "testing"

func TestStripControl(t *testing.T) {
	if got := stripControl("Donato Dozzy - K\n\x02m=k"); got != "Donato Dozzy - Km=k" {
		t.Errorf("got %q", got)
	}
}

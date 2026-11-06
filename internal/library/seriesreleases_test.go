package library

import (
	"testing"
	"time"

	"github.com/moodiness/polyfin/internal/stremio"
)

func TestSeriesToCome(t *testing.T) {
	now := time.Date(2026, 11, 6, 12, 0, 0, 0, time.UTC)
	for _, c := range []struct {
		name   string
		meta   stremio.Meta
		coming bool
	}{
		{"a day to come", stremio.Meta{Released: "2026-12-01T00:00:00.000Z"}, true},
		{"a day passed", stremio.Meta{Released: "2026-10-01"}, false},
		{"a year to come", stremio.Meta{Year: "2027"}, true},
		{"this year, no day", stremio.Meta{Year: "2026"}, false},
		{"a run from a past year", stremio.Meta{ReleaseInfo: "2019-2024"}, false},
		{"nothing told", stremio.Meta{}, false},
	} {
		if _, coming := seriesToCome(c.meta, now); coming != c.coming {
			t.Errorf("%s: %v, want %v", c.name, coming, c.coming)
		}
	}
}

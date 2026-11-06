package library

import (
	"time"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/stremio"
)

// Series wait for their first episode, as movies wait for their digital
// release (see releases.go): with Settings.WaitDigitalRelease on, a series
// listed by an addon is hidden until the day its listing tells it first
// airs. One whose listing tells no day stays listed: nothing tells it is
// not out.

// seriesToCome reports the day a series first airs, when its listing tells
// a day to come, or a year to come.
func seriesToCome(meta stremio.Meta, now time.Time) (time.Time, bool) {
	if day := exactDay(meta.Released); day != nil {
		return *day, day.After(now)
	}
	if year := metaYear(meta); year > now.Year() {
		return time.Date(year, time.January, 1, 0, 0, 0, 0, time.UTC), true
	}
	return time.Time{}, false
}

// heldSeries tells which of records the wait holds back for the user of
// v: the series of addons to come, by their item, with the day they air.
func (s *Service) heldSeries(v view, records []record) map[accounts.ID]time.Time {
	held := map[accounts.ID]time.Time{}
	now := s.now()
	for _, r := range records {
		if !v.waitRelease || r.Kind != KindSeries || r.Meta == nil {
			continue
		}
		if r.Addon != nil {
			if entry, ok := v.addon(*r.Addon); ok && own(entry.addon) {
				continue
			}
		}
		if day, coming := seriesToCome(*r.Meta, now); coming {
			held[r.ID] = day
		}
	}
	return held
}

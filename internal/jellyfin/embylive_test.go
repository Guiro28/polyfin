package jellyfin

import (
	"testing"
	"time"

	"github.com/moodiness/polyfin/internal/library"
)

// Emby for Android TV leaves the seconds out of its guide's dates.
func TestDatesWithoutSeconds(t *testing.T) {
	want := time.Date(2026, 11, 6, 10, 0, 0, 0, time.UTC)
	for _, text := range []string{"2026-11-06T10:00Z", "2026-11-06T11:00+01:00", "2026-11-06T10:00", "2026-11-06T10:00:00Z"} {
		if got, ok := parseTime(text); !ok || !got.Equal(want) {
			t.Errorf("%s: %v, %v", text, got, ok)
		}
	}
	if _, ok := parseTime("yesterday"); ok {
		t.Error("yesterday is read as a date")
	}
}

// A tag's channels are those of its group.
func TestChannelsOfATag(t *testing.T) {
	channels := []library.Item{{Name: "TF1", ChannelGroup: "France"}, {Name: "BBC", ChannelGroup: "UK"}, {Name: "M6", ChannelGroup: "France"}}
	got := channelsTagged(channels, []string{channelTagID("France")})
	if len(got) != 2 || got[0].Name != "TF1" || got[1].Name != "M6" {
		t.Errorf("France's channels: %+v", got)
	}
	if len(channels) != 3 {
		t.Error("the list given was changed")
	}
}

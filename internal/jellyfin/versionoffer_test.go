package jellyfin

import (
	"slices"
	"testing"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/library"
)

// offeredVersions are versions of a movie of the given heights, in that
// order, named after their place.
func offeredVersions(heights ...int) ([]library.Version, map[accounts.ID]int) {
	versions := make([]library.Version, len(heights))
	byID := map[accounts.ID]int{}
	for i, height := range heights {
		versions[i] = library.Version{ID: accounts.ID{byte(i + 1)}, Name: string(rune('a' + i)), Origin: library.Origin{Type: "movie"}}
		byID[versions[i].ID] = height
	}
	return versions, byID
}

func versionLetters(versions []library.Version) string {
	var text string
	for _, version := range versions {
		text += version.Name
	}
	return text
}

func TestBestVersionOfEachQuality(t *testing.T) {
	for _, c := range []struct {
		name    string
		heights []int
		group   int
		want    string
	}{
		// The first of each quality, in the rules' order, 4K first.
		{"each quality", []int{1080, 2160, 720, 1080, 2160}, 0, "bac"},
		// No 4K: the next lower quality first, never the same twice.
		{"no 4K", []int{1080, 720, 1080}, 0, "acb"},
		// No 720p: SD first, then the next higher quality.
		{"no 720p takes SD", []int{2160, 1080, 480}, 0, "abc"},
		{"no 720p nor SD takes higher", []int{2160, 1080, 1080}, 0, "abc"},
		// Fewer versions than qualities: each once.
		{"one version", []int{720}, 0, "a"},
		// Unknown heights come last.
		{"unknown", []int{0, 1080}, 0, "ba"},
		// A quality group of 1080p offers no 4K.
		{"group", []int{1080, 720, 1080}, 1080, "ab"},
	} {
		versions, heights := offeredVersions(c.heights...)
		got := versionLetters(byQuality(versions, heights, accounts.User{QualityGroup: c.group}))
		if got != c.want {
			t.Errorf("%s: offered %q, want %q", c.name, got, c.want)
		}
	}
}

func TestChannelsAreNotOfferedByQuality(t *testing.T) {
	versions, heights := offeredVersions(1080, 720, 1080)
	for i := range versions {
		versions[i].Origin.Type = "tv"
	}
	if got := byQuality(versions, heights, accounts.User{}); len(got) != 3 {
		t.Errorf("a channel offered %d versions", len(got))
	}
}

func TestFallbackQualities(t *testing.T) {
	for wanted, want := range map[int][]int{
		2160: {2160, 1080, 720, 480, 0},
		1080: {1080, 720, 480, 2160, 0},
		720:  {720, 480, 1080, 2160, 0},
	} {
		if got := fallbackQualities(wanted); !slices.Equal(got, want) {
			t.Errorf("%d falls back to %v, want %v", wanted, got, want)
		}
	}
}

func TestVersionLabelsGiveTheSizeAfterTheAddon(t *testing.T) {
	for _, c := range []struct{ name, size, want string }{
		{"[AD] Lumio · 4K · WEB-DL", "4,2 Go", "[AD] Lumio · 4,2 Go · 4K · WEB-DL"},
		{"Torrentio", "", "Torrentio"},
		{"Torrentio", "800 Mo", "Torrentio · 800 Mo"},
		// A name giving a size already keeps it as it is.
		{"Shelf · 1080p · 52 KB", "52 Ko", "Shelf · 1080p · 52 KB"},
		{"Torrentio · 💾 4.2 GB", "4,2 Go", "Torrentio · 💾 4.2 GB"},
	} {
		if got := versionLabel(c.name, c.size); got != c.want {
			t.Errorf("label of %q: %q, want %q", c.name, got, c.want)
		}
	}
	for _, c := range []struct {
		size     int64
		language string
		want     string
	}{
		{4_200_000_000, "fr", "4,2 Go"},
		{4_200_000_000, "en", "4.2 GB"},
		{800_000_000, "fr", "800 Mo"},
		{450_000_000, "en", "450 MB"},
		{1_050_000_000, "fr", "1,1 Go"},
		{3_000_000_000, "en", "3 GB"},
		{0, "fr", ""},
		{120_000_000_000, "en", "120 GB"},
	} {
		if got := sizeName(c.size, c.language); got != c.want {
			t.Errorf("size %d in %s: %q, want %q", c.size, c.language, got, c.want)
		}
	}
}

package jellyfin

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/library"
)

// Versions offered: with the server's VersionsOffered set to "qualities", a
// movie or an episode offers three versions, the best of each of 4K, 1080p
// and 720p in the rules' order, rather than every release the addons list.
// A quality none of its versions has is taken from the next lower quality
// first, then the next higher, never the same version twice. Whatever the
// setting, versions are named for version menus (see labeled).

// offeredQualities are the qualities offered, by height, in the order
// apps list them: the first is the one Play starts.
var offeredQualities = []int{2160, 1080, 720}

// qualityTiers are every quality a version may have, tallest first: 480
// stands for any lower definition.
var qualityTiers = []int{2160, 1080, 720, 480}

// qualityOf is the quality of a version of height, one of qualityTiers, 0
// when the height is not known.
func qualityOf(height int) int {
	switch {
	case height <= 0:
		return 0
	case height >= 1600:
		return 2160
	case height >= 900:
		return 1080
	case height >= 600:
		return 720
	}
	return 480
}

// titleVersion reports whether a version is a movie's or an episode's, the
// versions offered by quality and named with their size: not a channel's
// or a song's.
func titleVersion(version library.Version) bool {
	return version.Origin.Type == "movie" || version.Origin.Type == "series"
}

// byQuality keeps, of versions in the rules' order, the best of each of
// offeredQualities, for a user whose quality group leaves out the taller
// ones: for each, the first version of that quality, else of the next
// lower quality, else of the next higher one, else of an unknown height,
// not already offered.
func byQuality(versions []library.Version, heights map[accounts.ID]int, user accounts.User) []library.Version {
	if len(versions) == 0 || !titleVersion(versions[0]) {
		return versions
	}
	tiers := map[int][]library.Version{}
	for _, version := range versions {
		quality := qualityOf(heights[version.ID])
		tiers[quality] = append(tiers[quality], version)
	}
	used := map[accounts.ID]bool{}
	var offered []library.Version
	for _, wanted := range qualitySlots(user) {
		for _, quality := range fallbackQualities(wanted) {
			i := 0
			for i < len(tiers[quality]) && used[tiers[quality][i].ID] {
				i++
			}
			if i < len(tiers[quality]) {
				version := tiers[quality][i]
				used[version.ID] = true
				offered = append(offered, version)
				break
			}
		}
	}
	if len(offered) == 0 {
		return versions[:1]
	}
	return offered
}

// qualitySlots are the qualities offered to user, of offeredQualities those
// its quality group lets it play, in order: the n-th version offered by
// quality stands for the n-th.
func qualitySlots(user accounts.User) []int {
	return slices.DeleteFunc(slices.Clone(offeredQualities), func(quality int) bool { return !user.FitsGroup(quality) })
}

// fallbackQualities are the qualities a version of wanted quality is taken
// from, in turn: wanted, the lower ones, tallest first, the higher ones,
// lowest first, then an unknown height.
func fallbackQualities(wanted int) []int {
	order := []int{wanted}
	for _, quality := range qualityTiers {
		if quality < wanted {
			order = append(order, quality)
		}
	}
	for i := len(qualityTiers) - 1; i >= 0; i-- {
		if qualityTiers[i] > wanted {
			order = append(order, qualityTiers[i])
		}
	}
	return append(order, 0)
}

// labeled names a movie's or an episode's versions for version menus. Offered
// by quality, each is named after the quality it stands for, as Stremfin
// names them, nothing else: "4K", "1080p", "720p", even when it is taken
// from another quality. Otherwise each gives the addon's first line, its
// size, then the rest of its lines.
func (h *Handler) labeled(ctx context.Context, user accounts.User, versions []library.Version) []library.Version {
	if len(versions) == 0 || !titleVersion(versions[0]) {
		return versions
	}
	settings := h.Accounts.Settings()
	var slots []int
	if settings.VersionsOffered == "qualities" {
		slots = qualitySlots(user)
	}
	named := make([]library.Version, len(versions))
	for i, version := range versions {
		if i < len(slots) {
			version.Name = qualityName(slots[i])
		} else {
			version.Name = versionLabel(version.Name, sizeName(version.Size, settings.Language))
		}
		named[i] = version
	}
	return named
}

// byQualityNames reports whether version menus name a version by its
// quality only (see labeled), adding nothing to it.
func (h *Handler) byQualityNames(version library.Version) bool {
	return titleVersion(version) && h.Accounts.Settings().VersionsOffered == "qualities"
}

// namedSize matches a size a version's name gives already, as local
// folders and many addons write it: it is not given twice.
var namedSize = regexp.MustCompile(`(?i)\b\d+([.,]\d+)?\s?(KB|MB|GB|TB|Ko|Mo|Go|To)\b`)

// versionLabel joins the first part of name, as its addon names the
// stream, size and the rest of name, leaving out size when it is empty or
// name gives one already.
func versionLabel(name, size string) string {
	if namedSize.MatchString(name) {
		size = ""
	}
	first, rest, _ := strings.Cut(name, " · ")
	var parts []string
	for _, part := range []string{first, size, rest} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, " · ")
}

// qualityName names a quality as version menus show it.
func qualityName(quality int) string {
	switch quality {
	case 2160:
		return "4K"
	case 1080:
		return "1080p"
	case 720:
		return "720p"
	case 480:
		return "SD"
	}
	return ""
}

// sizeName is a file size in gigabytes, or megabytes under one, in the
// server language's words and decimal mark; empty when not known.
func sizeName(size int64, language string) string {
	if size <= 0 {
		return ""
	}
	french := strings.HasPrefix(strings.ToLower(language), "fr")
	unit, value := "GB", float64(size)/1e9
	if value < 1 {
		unit, value = "MB", float64(size)/1e6
	}
	if french {
		unit = map[string]string{"GB": "Go", "MB": "Mo"}[unit]
	}
	text := strings.TrimSuffix(fmt.Sprintf("%.1f", value), ".0")
	if value >= 100 {
		text = fmt.Sprintf("%.0f", value)
	}
	if french {
		text = strings.Replace(text, ".", ",", 1)
	}
	return text + " " + unit
}

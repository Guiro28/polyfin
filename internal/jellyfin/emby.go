package jellyfin

import (
	"cmp"
	"context"
	"net/http"
	"slices"
	"strings"
	"time"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/embyapp"
	"github.com/moodiness/polyfin/internal/mediasegments"
)

// Emby's own apps, such as Emby for Android TV, speak Jellyfin's API with a
// few differences, which this file answers for them alone (see package
// embyapp for how their requests are told).

// embyVersion is the version Emby's apps are told: they refuse a server
// that does not give an Emby 4 version of four numbers.
const embyVersion = "4.9.1.80"

// legacy reports whether r may authorize the legacy ways: always for an
// Emby app, which knows no other, else as the settings say.
func (h *Handler) legacy(r *http.Request) bool {
	return embyapp.In(r.Context()) || h.Accounts.Settings().LegacyAuthorization
}

// embyIdentity fills what c lacks of the app's identity from the headers,
// or query parameters, Emby's apps send it in apart from the authorization.
func embyIdentity(r *http.Request, c *credentials) {
	from := func(name string) string { return cmp.Or(r.Header.Get(name), query(r, name)) }
	c.Client = cmp.Or(c.Client, from("X-Emby-Client"))
	c.Device = cmp.Or(c.Device, from("X-Emby-Device-Name"))
	c.DeviceID = cmp.Or(c.DeviceID, from("X-Emby-Device-Id"))
	c.Version = cmp.Or(c.Version, from("X-Emby-Client-Version"))
}

// decodeSignIn reads the name and password of a sign-in: JSON, or the form
// Emby's apps post, whose fields match without regard to case, as JSON's
// do, and whose password is Pw, or Password from older ones.
func decodeSignIn(r *http.Request, name, password *string) error {
	if !strings.HasPrefix(strings.ToLower(r.Header.Get("Content-Type")), "application/x-www-form-urlencoded") {
		var body struct {
			Username string
			Pw       string
		}
		err := decode(r, &body)
		*name, *password = body.Username, body.Pw
		return err
	}
	r.Body = http.MaxBytesReader(nil, r.Body, maxBody)
	if err := r.ParseForm(); err != nil {
		return err
	}
	field := func(names ...string) string {
		for _, name := range names {
			for key, values := range r.PostForm {
				if strings.EqualFold(key, name) && len(values) > 0 && values[0] != "" {
					return values[0]
				}
			}
		}
		return ""
	}
	*name = field("Username", "UserName", "Name")
	*password = field("Pw", "Password")
	return nil
}

// embyPublicInfo is what Emby's apps are told of the server without
// signing in, as an Emby server tells it: an Emby version, and no product
// name, which would name Jellyfin.
func embyPublicInfo(local, name, id string) PublicSystemInfo {
	return PublicSystemInfo{
		LocalAddress:           local,
		LocalAddresses:         []string{local},
		RemoteAddresses:        []string{},
		ServerName:             name,
		Version:                embyVersion,
		Id:                     id,
		StartupWizardCompleted: true,
	}
}

// embyPing answers a ping as Emby does.
func embyPing(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte("Emby Server"))
}

// wakeOnLanInfo lists the network cards Wake-on-LAN could wake the server
// through: none. Emby for Android TV closes its "Who's watching?" screen
// on this answer, and opens its home before its licence check ends, about
// half a second, which Android 14 then blocks, as no window shows. An Emby
// app is answered later, as by an Emby server whose licence is cached.
func (h *Handler) wakeOnLanInfo(w http.ResponseWriter, r *http.Request) {
	if embyapp.In(r.Context()) {
		select {
		case <-time.After(2500 * time.Millisecond):
		case <-r.Context().Done():
			return
		}
	}
	writeJSON(w, http.StatusOK, []struct{}{})
}

// embyMarkersWait bounds how long an item waits for its skip segments
// before it is answered without them; the segments keep being read, and
// the next answer has them.
const embyMarkersWait = 2 * time.Second

// embyMarkers adds a movie's or an episode's intro and credits to its
// chapters as Emby's markers, which Emby's apps' skip buttons read instead
// of media segments: IntroStart and IntroEnd, and CreditsStart. The
// chapters are then numbered, with one at the start.
func (h *Handler) embyMarkers(ctx context.Context, dto *BaseItemDto, opened accounts.ID) {
	if (dto.Type != "Movie" && dto.Type != "Episode") || !h.Accounts.Settings().SkipButtons {
		return
	}
	user := callerFrom(ctx).User
	found := make(chan []mediasegments.Segment, 1)
	go func() {
		// Read to its end even when the answer stops waiting, for the next one.
		ctx := context.WithoutCancel(ctx)
		item, err := h.title(ctx, user, opened)
		if err != nil {
			found <- nil
			return
		}
		segments, _ := h.titleSegments(ctx, user, item, opened)
		found <- segments
	}()
	var segments []mediasegments.Segment
	select {
	case segments = <-found:
	case <-time.After(embyMarkersWait):
	case <-ctx.Done():
	}
	chapters := *dto.Chapters
	for _, segment := range segments {
		switch segment.Type {
		case mediasegments.Intro:
			chapters = append(chapters,
				ChapterInfo{StartPositionTicks: int64(segment.Start / 100), Name: "Intro", MarkerType: "IntroStart"},
				ChapterInfo{StartPositionTicks: int64(segment.End / 100), Name: "Intro", MarkerType: "IntroEnd"})
		case mediasegments.Outro:
			chapters = append(chapters,
				ChapterInfo{StartPositionTicks: int64(segment.Start / 100), Name: "Credits", MarkerType: "CreditsStart"})
		}
	}
	if len(chapters) == len(*dto.Chapters) {
		return
	}
	if !slices.ContainsFunc(chapters, func(c ChapterInfo) bool { return c.StartPositionTicks == 0 }) {
		chapters = append(chapters, ChapterInfo{Name: "Start"})
	}
	slices.SortStableFunc(chapters, func(a, b ChapterInfo) int { return cmp.Compare(a.StartPositionTicks, b.StartPositionTicks) })
	for i := range chapters {
		chapters[i].MarkerType = cmp.Or(chapters[i].MarkerType, "Chapter")
		chapters[i].ChapterIndex = &i
	}
	*dto.Chapters = chapters
}

// embySectionItems lists the titles of a home section Emby's server keeps
// for its apps (/Users/{userId}/Sections/{sectionId}/Items), which Polyfin
// has not: none, so that Emby for Android TV leaves the row out rather
// than showing it loading.
func (h *Handler) embySectionItems(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, QueryResult{Items: []BaseItemDto{}})
}

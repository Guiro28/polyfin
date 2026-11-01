package admin

import (
	"cmp"
	"context"
	_ "embed"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/moodiness/polyfin/internal/accounts"
	"github.com/moodiness/polyfin/internal/stremio"
)

// Profile pictures, as in Stremfin and Nuvio Manager: an administrator sets
// anyone's, and every user their own, from a picture sent or from
// Xperience's gallery of avatars, whose picture the server downloads. The
// apps get it as they get one sent from them (see accounts.Store.SetImage).

// xperienceCatalog is where Xperience lists its avatars; xperienceCDN is
// the only host their pictures are downloaded from.
const (
	xperienceCatalog = "https://xperience-app.com/api/avatars"
	xperienceGallery = "https://xperience-app.com/avatars"
	xperienceCDN     = "cdn.xperience-app.com"
	// xperienceKept is how long the catalog is kept once read.
	xperienceKept = 5 * time.Minute
	// xperienceWait bounds reading the catalog or a picture.
	xperienceWait = 25 * time.Second
)

// xperienceCopy is a copy of the catalog, answered when Xperience does not
// answer.
//
//go:embed xperience-avatars.json
var xperienceCopy []byte

// xperienceAvatar is an avatar of the gallery.
type xperienceAvatar struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	URL      string `json:"url"`
	Category string `json:"category"`
	order    int
}

// xperienceJSON is the gallery: its avatars, in its order, their
// categories, sorted, and whether it is the copy, Xperience not answering.
type xperienceJSON struct {
	Avatars    []xperienceAvatar `json:"avatars"`
	Categories []string          `json:"categories"`
	SourceURL  string            `json:"sourceUrl"`
	Copy       bool              `json:"copy"`
}

// xperienceCache keeps the catalog read last, and the copy once read.
var xperienceCache struct {
	mu      sync.Mutex
	catalog *xperienceJSON
	until   time.Time
	copy    *xperienceJSON
}

// xperienceURL is address when it is a picture of Xperience's CDN over
// https, empty otherwise.
func xperienceURL(address string) string {
	parsed, err := url.Parse(strings.TrimSpace(address))
	if err != nil || parsed.Scheme != "https" || parsed.Hostname() != xperienceCDN {
		return ""
	}
	return parsed.String()
}

// parseXperience reads the catalog as Xperience answers it: its avatars of
// a picture on its CDN, 5000 at most, in their order, then by name.
func parseXperience(body []byte) (*xperienceJSON, error) {
	var payload struct {
		Avatars []struct {
			ID        string `json:"id"`
			Slug      string `json:"slug"`
			Name      string `json:"name"`
			URL       string `json:"url"`
			Category  string `json:"category"`
			SortOrder *int   `json:"sort_order"`
		} `json:"avatars"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}
	var avatars []xperienceAvatar
	for i, row := range payload.Avatars {
		if i == 5000 {
			break
		}
		address := xperienceURL(row.URL)
		if address == "" {
			continue
		}
		avatar := xperienceAvatar{
			ID:       cmp.Or(row.ID, row.Slug, "avatar-"+strconv.Itoa(i)),
			Name:     truncate(cmp.Or(row.Name, row.Slug, "Avatar"), 160),
			URL:      address,
			Category: truncate(strings.TrimSpace(row.Category), 100),
			order:    i,
		}
		if row.SortOrder != nil {
			avatar.order = *row.SortOrder
		}
		avatars = append(avatars, avatar)
	}
	if len(avatars) == 0 {
		return nil, errors.New("no avatar in the catalog")
	}
	slices.SortStableFunc(avatars, func(a, b xperienceAvatar) int {
		return cmp.Or(cmp.Compare(a.order, b.order), strings.Compare(a.Name, b.Name))
	})
	var categories []string
	for _, avatar := range avatars {
		if !slices.Contains(categories, avatar.Category) {
			categories = append(categories, avatar.Category)
		}
	}
	// Avatars of no category come last.
	slices.SortFunc(categories, func(a, b string) int {
		if (a == "") != (b == "") {
			return cmp.Compare(len(b), len(a))
		}
		return strings.Compare(strings.ToLower(a), strings.ToLower(b))
	})
	return &xperienceJSON{Avatars: avatars, Categories: categories, SourceURL: xperienceGallery}, nil
}

func truncate(text string, size int) string {
	if runes := []rune(text); len(runes) > size {
		return string(runes[:size])
	}
	return text
}

// xperienceAvatars answers the gallery: the catalog read within
// xperienceKept, else read again, else the last one read, else the copy.
func (h *handler) xperienceAvatars(w http.ResponseWriter, r *http.Request) {
	xperienceCache.mu.Lock()
	defer xperienceCache.mu.Unlock()
	if xperienceCache.catalog != nil && time.Now().Before(xperienceCache.until) {
		writeJSON(w, http.StatusOK, xperienceCache.catalog)
		return
	}
	if catalog, err := readXperience(r.Context()); err == nil {
		xperienceCache.catalog, xperienceCache.until = catalog, time.Now().Add(xperienceKept)
		writeJSON(w, http.StatusOK, catalog)
		return
	} else {
		h.Logger.Info("Xperience's avatars could not be read: its copy is offered", "error", err)
	}
	if xperienceCache.catalog != nil {
		writeJSON(w, http.StatusOK, xperienceCache.catalog)
		return
	}
	if xperienceCache.copy == nil {
		catalog, err := parseXperience(xperienceCopy)
		if err != nil {
			h.internalError(w, r, err)
			return
		}
		catalog.Copy = true
		xperienceCache.copy = catalog
	}
	writeJSON(w, http.StatusOK, xperienceCache.copy)
}

// readXperience reads the catalog from Xperience.
func readXperience(ctx context.Context) (*xperienceJSON, error) {
	ctx, cancel := context.WithTimeout(ctx, xperienceWait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, xperienceCatalog, nil)
	if err != nil {
		return nil, err
	}
	request.Header.Set("Accept", "application/json")
	response, err := stremio.HTTPClient(true).Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d", response.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(response.Body, 16<<20))
	if err != nil {
		return nil, err
	}
	return parseXperience(body)
}

// imageTagJSON is a user's picture once changed: its tag, nil without one.
type imageTagJSON struct {
	ImageTag *string `json:"imageTag"`
}

// userImageTarget is the user whose picture a request changes: the user of
// the path, for an administrator, else the signed-in user.
func userImageTarget(w http.ResponseWriter, r *http.Request) (accounts.ID, bool) {
	if r.PathValue("id") != "" {
		return pathID(w, r, "id")
	}
	return sessionFrom(r.Context()).User.ID, true
}

// setUserImage replaces a user's picture with the one sent: a JPEG, PNG
// or WebP image in base64.
func (h *handler) setUserImage(w http.ResponseWriter, r *http.Request) {
	id, ok := userImageTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		Data string `json:"data"`
	}
	if !decodeUpTo(w, r, &body, accounts.MaxImageBytes/3*4+64<<10) {
		return
	}
	data, err := base64.StdEncoding.DecodeString(body.Data)
	if err != nil || len(data) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_image")
		return
	}
	h.keepUserImage(w, r, id, data)
}

// setXperienceImage makes a picture of Xperience's gallery a user's
// picture: the server downloads it from Xperience's CDN.
func (h *handler) setXperienceImage(w http.ResponseWriter, r *http.Request) {
	id, ok := userImageTarget(w, r)
	if !ok {
		return
	}
	var body struct {
		URL string `json:"url"`
	}
	if !decode(w, r, &body) {
		return
	}
	address := xperienceURL(body.URL)
	if address == "" {
		writeError(w, http.StatusBadRequest, "invalid_image_url")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), xperienceWait)
	defer cancel()
	request, err := http.NewRequestWithContext(ctx, http.MethodGet, address, nil)
	if err != nil {
		writeError(w, http.StatusBadRequest, "invalid_image_url")
		return
	}
	response, err := stremio.HTTPClient(true).Do(request)
	if err != nil {
		writeError(w, http.StatusBadGateway, "image_unreachable")
		return
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		writeError(w, http.StatusBadGateway, "image_unreachable")
		return
	}
	data, err := io.ReadAll(io.LimitReader(response.Body, accounts.MaxImageBytes+1))
	if err != nil {
		writeError(w, http.StatusBadGateway, "image_unreachable")
		return
	}
	if len(data) > accounts.MaxImageBytes {
		writeError(w, http.StatusBadRequest, "invalid_image")
		return
	}
	h.keepUserImage(w, r, id, data)
}

// keepUserImage keeps data as a user's picture, normalized, and answers
// its tag.
func (h *handler) keepUserImage(w http.ResponseWriter, r *http.Request, id accounts.ID, data []byte) {
	user, err := h.Accounts.SetImage(r.Context(), id, data)
	switch {
	case errors.Is(err, accounts.ErrInvalidImage):
		writeError(w, http.StatusBadRequest, "invalid_image")
	case errors.Is(err, accounts.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case err != nil:
		h.internalError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, imageTagJSON{ImageTag: imageTag(user)})
	}
}

// deleteUserImage removes a user's picture.
func (h *handler) deleteUserImage(w http.ResponseWriter, r *http.Request) {
	id, ok := userImageTarget(w, r)
	if !ok {
		return
	}
	user, err := h.Accounts.DeleteImage(r.Context(), id)
	switch {
	case errors.Is(err, accounts.ErrNotFound):
		writeError(w, http.StatusNotFound, "not_found")
	case err != nil:
		h.internalError(w, r, err)
	default:
		writeJSON(w, http.StatusOK, imageTagJSON{ImageTag: imageTag(user)})
	}
}

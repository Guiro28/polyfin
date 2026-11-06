package admin

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/moodiness/polyfin/internal/library"
)

// channelGroupImage is the address of the picture of the group of channels
// name, as apps ask for it, nil without one.
func (h *handler) channelGroupImage(name string) *string {
	if h.LibraryImages == nil || name == "" {
		return nil
	}
	id := library.ChannelGroupID(name)
	url, ok := h.LibraryImages.UploadedArtwork(id, "Primary")
	if !ok {
		return nil
	}
	address := "/Items/" + id.String() + "/Images/Primary?tag=" + library.ImageTag(url)
	return &address
}

// setChannelGroupImage keeps the picture sent, in base64, as the picture
// of the group of channels the body names, which Emby's apps show on its
// tag.
func (h *handler) setChannelGroupImage(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name string `json:"name"`
		Data string `json:"data"`
	}
	if !decodeUpTo(w, r, &body, library.MaxUploadedImageBytes/3*4+64<<10) {
		return
	}
	name := strings.TrimSpace(body.Name)
	data, err := base64.StdEncoding.DecodeString(body.Data)
	if name == "" || err != nil || len(data) == 0 {
		writeError(w, http.StatusBadRequest, "invalid_image")
		return
	}
	err = h.LibraryImages.UploadImage(r.Context(), library.ChannelGroupID(name), "Primary", data)
	if libraryImageError(w, err) {
		return
	}
	if err != nil {
		h.internalError(w, r, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]*string{"image": h.channelGroupImage(name)})
}

// deleteChannelGroupImage removes the picture of the group of channels
// the address names.
func (h *handler) deleteChannelGroupImage(w http.ResponseWriter, r *http.Request) {
	name := strings.TrimSpace(r.URL.Query().Get("name"))
	if name == "" {
		writeError(w, http.StatusBadRequest, "invalid_request")
		return
	}
	if err := h.LibraryImages.DeleteUploadedImage(r.Context(), library.ChannelGroupID(name), "Primary"); err != nil {
		h.internalError(w, r, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

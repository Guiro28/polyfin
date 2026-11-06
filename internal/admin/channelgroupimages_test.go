package admin

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"net/url"
	"strings"
	"testing"
)

// An administrator gives a group of channels a picture, which apps then
// load, and removes it; a member does not.
func TestChannelGroupPictures(t *testing.T) {
	t.Parallel()
	api := newTestAPI(t, 10)
	administrator := api.signedIn("administrator", true)
	member := api.signedIn("member", false)
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	sent := map[string]string{"name": "FR| CINEMA", "data": base64.StdEncoding.EncodeToString(picture.Bytes())}

	if status, _, _ := member.call(http.MethodPut, "/channel-groups/image", sent); status != http.StatusForbidden {
		t.Errorf("a member setting a group's picture: %d", status)
	}
	status, body, _ := administrator.call(http.MethodPut, "/channel-groups/image", sent)
	address, _ := body["image"].(string)
	if status != http.StatusOK || !strings.HasPrefix(address, "/Items/") {
		t.Fatalf("setting a group's picture: %d %v", status, body)
	}
	if status, body, _ := administrator.call(http.MethodPut, "/channel-groups/image", map[string]string{"name": "FR| CINEMA", "data": "bm90IGFuIGltYWdl"}); status != http.StatusBadRequest || body["error"] != "invalid_image" {
		t.Errorf("not a picture: %d %v", status, body)
	}
	if status, _, _ := administrator.call(http.MethodDelete, "/channel-groups/image?name="+url.QueryEscape("FR| CINEMA"), nil); status != http.StatusNoContent {
		t.Errorf("removing it: %d", status)
	}
	if status, body, _ := administrator.call(http.MethodDelete, "/channel-groups/image", nil); status != http.StatusBadRequest {
		t.Errorf("removing without a name: %d %v", status, body)
	}
}

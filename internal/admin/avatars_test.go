package admin

import (
	"bytes"
	"encoding/base64"
	"image"
	"image/png"
	"net/http"
	"testing"
)

// The copy of Xperience's gallery reads as the catalog does, and only
// pictures of its CDN are kept.
func TestXperienceCatalog(t *testing.T) {
	catalog, err := parseXperience(xperienceCopy)
	if err != nil || len(catalog.Avatars) < 100 || len(catalog.Categories) == 0 {
		t.Fatalf("the copy: %v avatars, %v categories, %v", len(catalog.Avatars), catalog.Categories, err)
	}
	for address, want := range map[string]bool{
		"https://cdn.xperience-app.com/avatars/nuvio/one.webp": true,
		"http://cdn.xperience-app.com/avatars/nuvio/one.webp":  false,
		"https://example.com/one.webp":                         false,
		"https://cdn.xperience-app.com.example.com/one.webp":   false,
	} {
		if got := xperienceURL(address) != ""; got != want {
			t.Errorf("%s: %v, want %v", address, got, want)
		}
	}
}

// An administrator sets and removes anyone's picture, and a member their
// own only; an address other than Xperience's CDN is refused.
func TestUserPictures(t *testing.T) {
	t.Parallel()
	api := newTestAPI(t, 10)
	administrator := api.signedIn("administrator", true)
	member := api.signedIn("member", false)
	var picture bytes.Buffer
	if err := png.Encode(&picture, image.NewRGBA(image.Rect(0, 0, 64, 64))); err != nil {
		t.Fatal(err)
	}
	sent := map[string]string{"data": base64.StdEncoding.EncodeToString(picture.Bytes())}

	users, err := api.store.Users(t.Context())
	if err != nil {
		t.Fatal(err)
	}
	memberID := ""
	for _, user := range users {
		if user.Name == "member" {
			memberID = user.ID.String()
		}
	}
	if status, body, _ := administrator.call(http.MethodPut, "/users/"+memberID+"/image", sent); status != http.StatusOK || body["imageTag"] == nil {
		t.Fatalf("an administrator setting a member's picture: %d %v", status, body)
	}
	if status, body, _ := administrator.call(http.MethodDelete, "/users/"+memberID+"/image", nil); status != http.StatusOK || body["imageTag"] != nil {
		t.Errorf("removing it: %d %v", status, body)
	}
	if status, body, _ := member.call(http.MethodPut, "/account/image", sent); status != http.StatusOK || body["imageTag"] == nil {
		t.Errorf("a member setting their own: %d %v", status, body)
	}
	if status, _, _ := member.call(http.MethodPut, "/users/"+memberID+"/image", sent); status != http.StatusForbidden {
		t.Errorf("a member through the users' route: %d", status)
	}
	if status, body, _ := member.call(http.MethodPut, "/account/image", map[string]string{"data": "bm90IGFuIGltYWdl"}); status != http.StatusBadRequest || body["error"] != "invalid_image" {
		t.Errorf("not a picture: %d %v", status, body)
	}
	if status, body, _ := member.call(http.MethodPost, "/account/image/xperience", map[string]string{"url": "https://example.com/one.webp"}); status != http.StatusBadRequest || body["error"] != "invalid_image_url" {
		t.Errorf("a picture elsewhere than Xperience: %d %v", status, body)
	}
}

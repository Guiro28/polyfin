package library

import (
	"crypto/sha256"

	"github.com/moodiness/polyfin/internal/accounts"
)

// ChannelGroupID names a group of channels, as its picture is kept among
// uploaded artwork and Emby's apps list it among their tags, by its name:
// groups of the same name are one, and a group renamed is another.
func ChannelGroupID(name string) accounts.ID {
	sum := sha256.Sum256([]byte("channel tag\x00" + name))
	var id accounts.ID
	copy(id[:], sum[:])
	return id
}

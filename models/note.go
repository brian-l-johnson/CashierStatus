package models

import (
	"regexp"
	"strings"

	"gorm.io/gorm"
)

// Note is one rotating message on the info board. ImageURL is always either
// empty or a server-local "/uploads/..." path — never an external URL. See
// ValidImageURL and the upload handler for why.
type Note struct {
	gorm.Model `json:"-"`
	ID         uint   `json:"id"`
	Message    string `json:"message"`
	ImageURL   string `json:"image_url"`
	Position   int    `json:"position"`
	Active     bool   `json:"active"`
	DwellSec   int    `json:"dwell_sec"` // 0 = use board default
}

const MaxMessageLen = 500

const (
	MinDwellSec = 5
	MaxDwellSec = 300
)

// imageURLPattern deliberately admits only paths this server generated itself.
// The upload handler names files with GenerateRandomString, whose charset is
// [0-9A-Za-z], so nothing legitimate needs a wider pattern.
var imageURLPattern = regexp.MustCompile(`^/uploads/[A-Za-z0-9]+\.(png|jpg|jpeg|gif|webp)$`)

// ValidImageURL reports whether a note's image reference is safe to store. It
// is the only thing standing between an admin form and a kiosk browser pointed
// at an arbitrary host, so it is an allow-list: empty, or one of our own upload
// paths. External URLs and javascript:/data: schemes are all rejected by
// failing to match, and so is any path containing traversal characters.
func ValidImageURL(url string) bool {
	return url == "" || imageURLPattern.MatchString(url)
}

// ValidMessage reports whether a trimmed message is an acceptable length.
func ValidMessage(message string) bool {
	trimmed := strings.TrimSpace(message)
	return trimmed != "" && len(trimmed) <= MaxMessageLen
}

// ClampDwellSec normalizes a per-note dwell time. Zero means "use the board
// default" and is passed through; anything else is pinned into a range that
// keeps a slide readable without stalling the rotation.
func ClampDwellSec(dwell int) int {
	if dwell <= 0 {
		return 0
	}
	if dwell < MinDwellSec {
		return MinDwellSec
	}
	if dwell > MaxDwellSec {
		return MaxDwellSec
	}
	return dwell
}

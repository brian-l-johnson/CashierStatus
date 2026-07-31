package models

import (
	"strings"
	"testing"
)

func TestValidImageURL(t *testing.T) {
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"empty is allowed", "", true},
		{"generated png path", "/uploads/aB3xY9zQ1mN4pR7s.png", true},
		{"jpg", "/uploads/abc123.jpg", true},
		{"jpeg", "/uploads/abc123.jpeg", true},
		{"gif", "/uploads/abc123.gif", true},
		{"webp", "/uploads/abc123.webp", true},

		{"external https url", "https://evil.example/x.png", false},
		{"protocol relative url", "//evil.example/x.png", false},
		{"javascript scheme", "javascript:alert(1)", false},
		{"data uri", "data:image/png;base64,AAAA", false},
		{"path traversal", "/uploads/../../etc/passwd", false},
		{"traversal with valid suffix", "/uploads/../secret.png", false},
		{"nested path", "/uploads/sub/dir/x.png", false},
		{"wrong prefix", "/static/x.png", false},
		{"relative path", "uploads/x.png", false},
		{"no extension", "/uploads/abc123", false},
		{"disallowed extension", "/uploads/abc123.svg", false},
		{"double extension", "/uploads/abc123.png.html", false},
		{"hyphen in name", "/uploads/abc-123.png", false},
		{"trailing newline", "/uploads/abc123.png\n", false},
		{"leading whitespace", " /uploads/abc123.png", false},
		{"query string appended", "/uploads/abc123.png?x=1", false},
		{"uppercase extension", "/uploads/abc123.PNG", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := ValidImageURL(tc.url); got != tc.want {
				t.Errorf("ValidImageURL(%q) = %v, want %v", tc.url, got, tc.want)
			}
		})
	}
}

func TestClampDwellSec(t *testing.T) {
	cases := []struct {
		in   int
		want int
	}{
		{0, 0},
		{-10, 0},
		{1, MinDwellSec},
		{4, MinDwellSec},
		{5, 5},
		{20, 20},
		{300, 300},
		{301, MaxDwellSec},
		{99999, MaxDwellSec},
	}

	for _, tc := range cases {
		if got := ClampDwellSec(tc.in); got != tc.want {
			t.Errorf("ClampDwellSec(%d) = %d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestValidMessage(t *testing.T) {
	if ValidMessage("") {
		t.Error("empty message should be rejected")
	}
	if ValidMessage("   \n\t ") {
		t.Error("whitespace-only message should be rejected")
	}
	if !ValidMessage("No outside food or drink") {
		t.Error("ordinary message should be accepted")
	}
	if !ValidMessage(strings.Repeat("a", MaxMessageLen)) {
		t.Error("message at the length limit should be accepted")
	}
	if ValidMessage(strings.Repeat("a", MaxMessageLen+1)) {
		t.Error("message over the length limit should be rejected")
	}
}

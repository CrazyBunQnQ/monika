package builtin

import "testing"

func TestDetectImageMime(t *testing.T) {
	// Minimal real-ish bytes: just enough for the magic check.
	png := append([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}, []byte("rest")...)
	jpeg := []byte{0xFF, 0xD8, 0xFF, 0xE0, 0, 0x10, 'J', 'F', 'I', 'F'}
	gif := []byte("GIF89a-rest-of-gif")
	webp := append([]byte("RIFF"), []byte{0, 0, 0, 0}...)
	webp = append(webp, []byte("WEBPVP8 ")...)

	cases := []struct {
		name string
		path string
		data []byte
		want string
	}{
		{"png by magic", "/tmp/x.png", png, "image/png"},
		{"png by extension only rejected", "/tmp/x.png", []byte("not png"), ""},
		{"jpeg", "/tmp/x.jpg", jpeg, "image/jpeg"},
		{"gif", "/tmp/x.gif", gif, "image/gif"},
		{"webp", "/tmp/x.webp", webp, "image/webp"},
		{"plain text rejected", "/tmp/x.png", []byte("hello"), ""},
		{"empty rejected", "/tmp/x.png", nil, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := detectImageMime(tc.path, tc.data); got != tc.want {
				t.Errorf("got %q, want %q", got, tc.want)
			}
		})
	}
}

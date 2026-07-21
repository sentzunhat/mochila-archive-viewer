package archive

import "testing"

func TestMediaEntryToken(t *testing.T) {
	cases := []struct {
		entry string
		want  string
	}{
		{
			entry: "chat_media/2019-04-01_b~EiQSFXJsVjBNaGpvYWx0bmMzNnpPSFdvdxoAGgAyAQNIAlAEYAE.jpg",
			want:  "b~EiQSFXJsVjBNaGpvYWx0bmMzNnpPSFdvdxoAGgAyAQNIAlAEYAE",
		},
		{
			// No date prefix — falls back to the whole basename minus extension.
			entry: "chat_media/unknown_token.jpg",
			want:  "unknown_token",
		},
		{
			entry: "memories/2026-06-07_b~EiASFXNacllMb0NkUlVYaUdweWVVdnFVdTIBVUgCUARgAQ.mp4",
			want:  "b~EiASFXNacllMb0NkUlVYaUdweWVVdnFVdTIBVUgCUARgAQ",
		},
	}
	for _, tc := range cases {
		if got := mediaEntryToken(tc.entry); got != tc.want {
			t.Errorf("mediaEntryToken(%q) = %q, want %q", tc.entry, got, tc.want)
		}
	}
}

package tunnel

import "testing"

func TestParseURL(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "standard quick tunnel URL",
			input: `2024/01/15 10:00:02 INF |  https://random-words-here.trycloudflare.com  |`,
			want:  "https://random-words-here.trycloudflare.com",
		},
		{
			name:  "URL in connection registered line",
			input: `INF Registered tunnel connection connIndex=0 url=https://my-tunnel-abc123.trycloudflare.com`,
			want:  "https://my-tunnel-abc123.trycloudflare.com",
		},
		{
			name:  "URL with multiple dashes",
			input: `https://able-similarly-monster-configured.trycloudflare.com`,
			want:  "https://able-similarly-monster-configured.trycloudflare.com",
		},
		{
			name:  "no match - different domain",
			input: `https://example.com/some/path`,
			want:  "",
		},
		{
			name:  "no match - empty line",
			input: ``,
			want:  "",
		},
		{
			name:  "no match - random log line",
			input: `2024/01/15 10:00:00 INF Starting tunnel`,
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseURL(tt.input)
			if got != tt.want {
				t.Errorf("ParseURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

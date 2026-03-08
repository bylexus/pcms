package webserver

import "testing"

func TestNormalizeFileLookupRoute(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantRoute string
		wantOK    bool
	}{
		{name: "root", in: "/", wantRoute: "/", wantOK: true},
		{name: "file", in: "/robots.txt", wantRoute: "/robots.txt", wantOK: true},
		{name: "strict trailing slash", in: "/robots.txt/", wantRoute: "", wantOK: false},
		{name: "double slash trailing", in: "//robots.txt//", wantRoute: "", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotRoute, gotOK := normalizeFileLookupRoute(tt.in)
			if gotOK != tt.wantOK {
				t.Fatalf("normalizeFileLookupRoute(%q) ok = %v, want %v", tt.in, gotOK, tt.wantOK)
			}
			if gotRoute != tt.wantRoute {
				t.Fatalf("normalizeFileLookupRoute(%q) route = %q, want %q", tt.in, gotRoute, tt.wantRoute)
			}
		})
	}
}

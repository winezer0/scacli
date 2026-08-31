package git

import "testing"

func TestNewRepository(t *testing.T) {
	tests := []struct {
		name   string
		url    string
		branch string
		want   Repository
	}{
		{name: "ssh", url: "git@git.example.com:ott/iptv-epg.git", branch: "master", want: Repository{URL: "git@git.example.com:ott/iptv-epg.git", Branch: "master", Name: "ott_iptv-epg"}},
		{name: "https", url: "https://github.com/org/project.git", want: Repository{URL: "https://github.com/org/project.git", Name: "org_project"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NewRepository(test.url, test.branch); got != test.want {
				t.Fatalf("repository = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestRuntimeName(t *testing.T) {
	got := RuntimeName(NewRepository("git@git.example.com:ott/project.git", "feature/test"))
	if got != "ott_project@feature_test" {
		t.Fatalf("runtime name = %q, want %q", got, "ott_project@feature_test")
	}
}

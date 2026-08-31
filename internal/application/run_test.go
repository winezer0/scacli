package application

import (
	"testing"

	"scacli/internal/git"
	"scacli/internal/tasks"
)

func TestLoadRepositories(t *testing.T) {
	tests := []struct {
		name   string
		opts   Options
		config []tasks.Task
		want   []string
	}{
		{name: "command line overrides config", opts: Options{GitURLs: []string{"git@example.com:org/one.git master", "git@example.com:org/one.git master", "git@example.com:org/two.git"}}, want: []string{"org_one@master", "org_two@default"}},
		{name: "config repositories", config: []tasks.Task{{URL: "https://example.com/org/one.git", Branch: "dev"}}, want: []string{"org_one@dev"}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := loadRepositories(test.opts, test.config)
			if err != nil {
				t.Fatalf("load repositories: %v", err)
			}
			if len(got) != len(test.want) {
				t.Fatalf("repositories = %#v, want %v", got, test.want)
			}
			for index, repository := range got {
				if gotName := git.RuntimeName(repository); gotName != test.want[index] {
					t.Fatalf("repository %d name = %q, want %q", index, gotName, test.want[index])
				}
			}
		})
	}
}

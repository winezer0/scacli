package tasks

// Task identifies one Git repository and its optional branch or tag.
type Task struct {
	URL    string `yaml:"url" json:"url"`
	Branch string `yaml:"branch,omitempty" json:"branch,omitempty"`
}

// Config contains the repository scan task list.
type Config struct {
	Tasks []Task `yaml:"tasks" json:"tasks"`
}

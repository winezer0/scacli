package application

// Options contains command-line inputs and runtime overrides.
type Options struct {
	ConfigFile   string   `short:"c" long:"config" description:"Configuration file (default: config.yaml)"`
	TaskFile     string   `short:"T" long:"task" description:"Task file (default: task.yaml)"`
	Engine       string   `long:"engine" choice:"murphysec" choice:"opensca" choice:"osv-scanner" description:"SCA engine"`
	GitURLs      []string `short:"g" long:"git" description:"Git repository URL; repeatable"`
	GitFile      string   `short:"f" long:"git-file" description:"File containing one repository per line"`
	OutputDir    string   `short:"o" long:"output" description:"Report output directory"`
	Formats      []string `short:"F" long:"format" description:"Report formats: json,html,md; repeatable or comma-separated"`
	Token        string   `short:"t" long:"token" description:"SCA token (overrides SCACLI_TOKEN)"`
	Murphysec    string   `short:"m" long:"murphysec" description:"MurphySec executable; default search: PATH, app directory, app/bin"`
	OpenSCA      string   `long:"opensca" description:"OpenSCA executable; default search: PATH, app directory, app/bin"`
	OSVScanner   string   `long:"osv-scanner" description:"OSV-Scanner executable; default search: PATH, app directory, app/bin"`
	SSHKey       string   `short:"k" long:"ssh-key" description:"SSH private key used by Git"`
	GitUser      string   `long:"git-user" description:"HTTPS Git username"`
	GitPassword  string   `long:"git-password" description:"HTTPS Git password/token (overrides GIT_PASSWORD)"`
	Branch       string   `short:"b" long:"branch" description:"Branch or tag applied to every repository"`
	CloneTimeout int      `long:"clone-timeout" description:"Git clone timeout in seconds"`
	ScanTimeout  int      `long:"scan-timeout" description:"SCA timeout in seconds"`
	Version      bool     `long:"version" description:"Print version"`
}

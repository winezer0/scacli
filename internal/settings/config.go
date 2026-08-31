package settings

// ScanConfig contains scanner selection and scanner-specific executable settings.
type ScanConfig struct {
	Engine      string `yaml:"engine"`
	Token       string `yaml:"token"`
	Murphysec   string `yaml:"murphysec"`
	OpenSCA     string `yaml:"opensca"`
	OSVScanner  string `yaml:"osv_scanner"`
	ScanTimeout int    `yaml:"scan_timeout"`
}

// GitConfig contains Git authentication and clone settings.
type GitConfig struct {
	SSHKey       string `yaml:"ssh_key"`
	GitUser      string `yaml:"user"`
	GitPassword  string `yaml:"password"`
	CloneTimeout int    `yaml:"clone_timeout"`
}

// ReportConfig contains report output settings.
type ReportConfig struct {
	OutputDir string `yaml:"output"`
}

// Config is the domain configuration aggregate used by the application.
type Config struct {
	Scan   ScanConfig   `yaml:"scan"`
	Git    GitConfig    `yaml:"git"`
	Report ReportConfig `yaml:"report"`
}

package version

type VersionInfo struct {
	BuildDate string `json:"buildDate" yaml:"buildDate"`
	Commit    string `json:"commit" yaml:"commit"`
	Version   string `json:"version" yaml:"version"`
}

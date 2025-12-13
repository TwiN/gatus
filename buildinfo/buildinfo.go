package buildinfo

var version = "dev"
var commitHash = "unknown"
var time = "unknown"

type BuildInfo struct {
	Version    string
	CommitHash string
	Time       string
}

func Get() BuildInfo {
	return BuildInfo{
		Version:    version,
		CommitHash: commitHash,
		Time:       time,
	}
}

package build

var version = "dev"
var commitHash = "unknown"
var time = "unknown"

type BuildInfo struct {
	Version    string
	CommitHash string
	Time       string
}

func GetBuildInfo() BuildInfo {
	return BuildInfo{
		Version:    version,
		CommitHash: commitHash,
		Time:       time,
	}
}

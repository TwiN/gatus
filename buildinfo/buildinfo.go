package buildinfo

var version = "dev"
var commitHash = "unknown"
var date = "unknown"

type BuildInfo struct {
	Version    string
	CommitHash string
	Date       string
}

func Get() BuildInfo {
	return BuildInfo{
		Version:    version,
		CommitHash: commitHash,
		Date:       date,
	}
}

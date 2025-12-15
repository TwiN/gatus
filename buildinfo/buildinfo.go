package buildinfo

import (
	"runtime"
	"runtime/debug"
)

const (
	defaultVersion  = "development"
	defaultRevision = "unknown"
	defaultDate     = "unknown"
)

var (
	version  string
	revision string
	date     string

	buildInfo = SetBuildInfo(GetDefault())
)

type BuildInfo struct {
	Version   string
	Revision  string
	Date      string
	GoVersion string
	Dirty     bool
}

func SetEmbedded(info *BuildInfo) {
	rawInfo, ok := debug.ReadBuildInfo()
	if !ok {
		info.Revision += "-buildinfo-err"
	}
	for _, s := range rawInfo.Settings {
		switch s.Key {
		case "vcs.revision":
			info.Revision = s.Value
		case "vcs.time":
			info.Date = s.Value
		case "vcs.modified":
			info.Dirty = s.Value == "true"
		}
	}
}

func SetLdflags(info *BuildInfo) {
	if len(version) > 0 {
		info.Version = version
	}
	if len(revision) > 0 {
		info.Revision = revision
	}
	if len(date) > 0 {
		info.Date = date
	}
	if len(info.GoVersion) == 0 {
		info.GoVersion = runtime.Version()
	}
}

func GetDefault() BuildInfo {
	return BuildInfo{
		Version:   defaultVersion,
		Revision:  defaultRevision,
		Date:      defaultDate,
		GoVersion: runtime.Version(),
		Dirty:     false,
	}
}

func SetBuildInfo(info BuildInfo) BuildInfo {
	SetEmbedded(&info)
	SetLdflags(&info)

	if info.Dirty {
		info.Revision += "-dirty"
	}
	return info
}

func Get() BuildInfo {
	return buildInfo
}

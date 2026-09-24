package buildinfo

import (
	"runtime"
	"runtime/debug"
)

const (
	defaultVersion      = "development"
	defaultRevision     = "unknown"
	defaultRevisionDate = "unknown"
)

var (
	version      string
	revision     string
	revisionDate string

	buildInfo = SetBuildInfo(GetDefault())
)

type BuildInfo struct {
	Version      string
	Revision     string
	RevisionDate string
	GoVersion    string
	Dirty        bool
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
			info.RevisionDate = s.Value
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
	if len(revisionDate) > 0 {
		info.RevisionDate = revisionDate
	}
	if len(info.GoVersion) == 0 {
		info.GoVersion = runtime.Version()
	}
}

func GetDefault() BuildInfo {
	return BuildInfo{
		Version:      defaultVersion,
		Revision:     defaultRevision,
		RevisionDate: defaultRevisionDate,
		GoVersion:    runtime.Version(),
		Dirty:        false,
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

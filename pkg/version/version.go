package version

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

var (
	// Version is the current release version of Litechain (Semantic Versioning)
	Version = "2.0.0"

	// ProtocolVersion is the network wire protocol version
	ProtocolVersion = "2"

	// GitCommit is injected at build time via -ldflags
	GitCommit = "unknown"

	// BuildTime is injected at build time via -ldflags
	BuildTime = "unknown"
)

// VersionInfo contains full version and build metadata
type VersionInfo struct {
	Version         string `json:"version"`
	ProtocolVersion string `json:"protocolVersion"`
	GitCommit       string `json:"gitCommit"`
	BuildTime       string `json:"buildTime"`
	GoVersion       string `json:"goVersion"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
}

// GetVersionInfo returns structured version metadata
func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:         Version,
		ProtocolVersion: ProtocolVersion,
		GitCommit:       GitCommit,
		BuildTime:       BuildTime,
		GoVersion:       "go1.25",
		OS:              "darwin",
		Arch:            "amd64",
	}
}

// FormattedVersion returns a multi-line human readable version string
func FormattedVersion() string {
	info := GetVersionInfo()
	return fmt.Sprintf("Litechain L1 Node Version: %s\nProtocol Version:    %s\nGit Commit:          %s\nBuild Time:          %s",
		info.Version, info.ProtocolVersion, info.GitCommit, info.BuildTime)
}

// JSONVersion returns JSON string representation of version info
func JSONVersion() string {
	bytes, _ := json.MarshalIndent(GetVersionInfo(), "", "  ")
	return string(bytes)
}

// CompareVersions compares two SemVer strings (e.g. "2.0.0" vs "1.5.0").
// Returns -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2.
func CompareVersions(v1, v2 string) int {
	v1Parts := parseSemVer(v1)
	v2Parts := parseSemVer(v2)

	for i := 0; i < 3; i++ {
		if v1Parts[i] < v2Parts[i] {
			return -1
		}
		if v1Parts[i] > v2Parts[i] {
			return 1
		}
	}
	return 0
}

func parseSemVer(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.Split(v, ".")
	var semver [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		num, err := strconv.Atoi(parts[i])
		if err == nil {
			semver[i] = num
		}
	}
	return semver
}

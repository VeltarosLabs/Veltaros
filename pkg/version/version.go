package version

import "runtime"

// These are intended to be set at build time via -ldflags.
// Example:
// go build -ldflags "-X github.com/VeltarosLabs/Veltaros/pkg/version.Version=0.1.0 -X github.com/VeltarosLabs/Veltaros/pkg/version.Commit=$(git rev-parse --short HEAD)"
var (
	Version = "0.1.0"
	Commit  = "dev"
)

type Info struct {
	Version   string `json:"version"`
	Commit    string `json:"commit"`
	GoVersion string `json:"goVersion"`
	Platform  string `json:"platform"`
}

func Get() Info {
	return Info{
		Version:   Version,
		Commit:    Commit,
		GoVersion: runtime.Version(),
		Platform:  runtime.GOOS + "/" + runtime.GOARCH,
	}
}

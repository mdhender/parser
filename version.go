// Cooyright (c) 2026 Michael D Henderson

package parser

import (
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major:      1,
		Minor:      1,
		Patch:      0,
		PreRelease: "",
	}
)

func Version() semver.Version {
	return version
}

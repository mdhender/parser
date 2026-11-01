// Cooyright (c) 2026 Michael D Henderson

package parser

import (
	"github.com/maloquacious/semver"
)

var (
	version = semver.Version{
		Major:      1,
		Minor:      2,
		Patch:      1,
		PreRelease: "",
	}
)

func Version() semver.Version {
	return version
}

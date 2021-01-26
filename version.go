package zeus

import (
	"fmt"

	"github.com/blang/semver"
)

const versionDevelopment = "development"

var ZEUS_VERSION = versionDevelopment

func VersionAreCompatible(a, b string) (bool, error) {
	if a == versionDevelopment || b == versionDevelopment {
		return true, nil
	}
	av, err := semver.ParseTolerant(a)
	if err != nil {
		return false, fmt.Errorf("Invalid version '%s': %s", a, err)
	}
	bv, err := semver.ParseTolerant(b)
	if err != nil {
		return false, fmt.Errorf("Invalid version '%s': %s", b, err)
	}
	if av.Major == 0 {
		return bv.Major == 0 && av.Minor == bv.Minor, nil
	}
	return av.Major == bv.Major, nil
}

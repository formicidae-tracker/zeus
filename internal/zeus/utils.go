package zeus

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/blang/semver"
)

//go:generate go run generate_version.go $VERSION

func filenameWithSuffix(fpath string, iter uint) string {
	res := strings.TrimSuffix(fpath, filepath.Ext(fpath))
	curIter, _ := strconv.Atoi(strings.TrimPrefix(filepath.Ext(res), "."))
	if curIter != 0 {
		res = strings.TrimSuffix(res, filepath.Ext(res))
	}

	if iter == 0 {
		return res + filepath.Ext(fpath)
	}

	return res + "." + strconv.FormatUint(uint64(iter), 10) + filepath.Ext(fpath)
}

func CreateFileWithoutOverwrite(fpath string) (*os.File, string, error) {
	iter := uint(0)
	for {
		toTest := filenameWithSuffix(fpath, iter)
		if _, err := os.Stat(toTest); os.IsNotExist(err) {
			f, err := os.Create(toTest)
			return f, toTest, err
		}
		iter += 1
	}
}

// Tests if two version strings are compatible.
func VersionAreCompatible(a, b string) (bool, error) {
	if a == "development" || b == "development" {
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

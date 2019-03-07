package dieu

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

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

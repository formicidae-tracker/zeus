package zeus

import (
	"io/ioutil"
	"os"
	"path/filepath"

	. "gopkg.in/check.v1"
)

type UtilsSuite struct {
	tmpDir string
}

var _ = Suite(&UtilsSuite{})

func (s *UtilsSuite) SetUpSuite(c *C) {
	var err error
	s.tmpDir, err = ioutil.TempDir("", "dieu-utils-tests")

	c.Check(err, IsNil)
}

func (s *UtilsSuite) TearDownSuite(c *C) {
	c.Check(os.RemoveAll(s.tmpDir), IsNil)
}

func (s *UtilsSuite) TestNameSuffix(c *C) {
	testdata := []struct {
		Base     string
		i        uint
		Expected string
	}{
		{"out.txt", 0, "out.txt"},
		{"bar.foo.2.txt", 3, "bar.foo.3.txt"},
		{"../some/path/out.42.txt", 2, "../some/path/out.2.txt"},
		{"../some/path/out.42.txt", 0, "../some/path/out.txt"},
	}

	for _, d := range testdata {
		c.Check(filenameWithSuffix(d.Base, d.i), Equals, d.Expected)
	}
}

func (s *UtilsSuite) TestCreateWithoutOverwrite(c *C) {
	files := []string{"out.txt", "out.1.txt", "out.3.txt"}

	for _, f := range files {
		ff, err := os.Create(filepath.Join(s.tmpDir, f))
		c.Assert(err, IsNil)
		defer ff.Close()
	}

	ff, name, err := CreateFileWithoutOverwrite(filepath.Join(s.tmpDir, files[0]))
	c.Check(err, IsNil)
	defer ff.Close()
	c.Assert(name, Equals, filepath.Join(s.tmpDir, "out.2.txt"))
	_, err = os.Stat(name)
	c.Check(err, IsNil)

}

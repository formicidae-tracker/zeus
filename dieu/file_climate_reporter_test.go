package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"git.tuleu.science/fort/dieu"
	. "gopkg.in/check.v1"
)

type FileClimateReporterSuite struct {
	TmpDir string
}

func (s *FileClimateReporterSuite) SetUpSuite(c *C) {
	var err error
	s.TmpDir, err = ioutil.TempDir("", "file-report-notifier")
	c.Assert(err, IsNil)

}

func (s *FileClimateReporterSuite) TearDownSuite(c *C) {
	c.Assert(os.RemoveAll(s.TmpDir), IsNil)
}

var _ = Suite(&FileClimateReporterSuite{})

func (s *FileClimateReporterSuite) TestFileNameDoesNotOverwite(c *C) {
	_, name1, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"))
	c.Check(err, IsNil)
	_, name2, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"))

	c.Check(name1, Equals, filepath.Join(s.TmpDir, "test.txt"))
	c.Check(name2, Equals, filepath.Join(s.TmpDir, "test.1.txt"))
}

func (s *FileClimateReporterSuite) TestFileNameWriting(c *C) {
	n, fname, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"))
	c.Assert(err, IsNil)

	fn := n.(*fileClimateReporter)

	cr := dieu.ClimateReport{
		Humidity:     50,
		Temperatures: [4]dieu.Temperature{21, 21, 21, 21},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		n.Report()
		wg.Done()
	}()

	for i := 0; i < 4; i++ {
		cr.Time = fn.Start.Add(time.Duration(i*333) * time.Millisecond)
		n.C() <- cr
	}
	close(n.C())
	wg.Wait()

	data, err := ioutil.ReadFile(fname)
	c.Assert(err, IsNil)

	c.Check(string(data), Equals, fmt.Sprintf(`# Starting date %s
# Time(ms) Relative Humidity (%%) Temperature (°C) Temperature (°C) Temperature (°C) Temperature (°C)
0 50.00 21.00 21.00 21.00 21.00
333 50.00 21.00 21.00 21.00 21.00
666 50.00 21.00 21.00 21.00 21.00
999 50.00 21.00 21.00 21.00 21.00
`, fn.Start))

}

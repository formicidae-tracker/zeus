package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/formicidae-tracker/zeus"
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
	fn, fname, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"))
	c.Assert(err, IsNil)

	cr := zeus.ClimateReport{
		Humidity:     50,
		Temperatures: [4]zeus.Temperature{21, 21, 21, 21},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		fn.Report()
		wg.Done()
	}()

	for i := 0; i < 4; i++ {
		cr.Time = fn.(*fileClimateReporter).Start.Add(time.Duration(i*333) * time.Millisecond)
		fn.ReportChannel() <- cr
	}
	close(fn.ReportChannel())
	wg.Wait()

	data, err := ioutil.ReadFile(fname)
	c.Assert(err, IsNil)

	c.Check(string(data), Equals, fmt.Sprintf(`# Starting date %s
# Time(ms) Relative Humidity (%%) Temperature (°C) Temperature (°C) Temperature (°C) Temperature (°C)
0 50.00 21.00 21.00 21.00 21.00
333 50.00 21.00 21.00 21.00 21.00
666 50.00 21.00 21.00 21.00 21.00
999 50.00 21.00 21.00 21.00 21.00
`, fn.(*fileClimateReporter).Start))

}

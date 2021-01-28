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
	_, name1, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"), 0)
	c.Check(err, IsNil)
	_, name2, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"), 0)

	c.Check(name1, Equals, filepath.Join(s.TmpDir, "test.txt"))
	c.Check(name2, Equals, filepath.Join(s.TmpDir, "test.1.txt"))
}

func (s *FileClimateReporterSuite) TestFileNameWriting(c *C) {
	fn, fname, err := NewFileClimateReporter(filepath.Join(s.TmpDir, "test.txt"), 3)
	c.Assert(err, IsNil)

	cr := zeus.ClimateReport{
		Humidity:     50,
		Temperatures: []zeus.Temperature{21, 21, 21, 21},
	}
	wg := sync.WaitGroup{}
	wg.Add(1)
	ready := make(chan struct{})
	go func() {
		fn.Report(ready)
		wg.Done()
	}()
	<-ready

	for i := 0; i < 4; i++ {
		cr.Time = fn.(*fileClimateReporter).Start.Add(time.Duration(i*333) * time.Millisecond)
		fn.ReportChannel() <- cr
	}
	close(fn.ReportChannel())
	wg.Wait()

	data, err := ioutil.ReadFile(fname)
	c.Assert(err, IsNil)

	c.Check(string(data), Equals, fmt.Sprintf(`# Starting date %s
# Time (ms) Relative Humidity (%%) Temperature (°C) Aux 1 (°C) Aux 2 (°C) Aux 3 (°C)
0 50.00 21.00 21.00 21.00 21.00
333 50.00 21.00 21.00 21.00 21.00
666 50.00 21.00 21.00 21.00 21.00
999 50.00 21.00 21.00 21.00 21.00
`, fn.(*fileClimateReporter).Start.Format(time.RFC3339Nano)))

}

func (s *FileClimateReporterSuite) TestFileReading(c *C) {
	tmpdir, err := ioutil.TempDir("", "zeus-read-season-file")
	c.Assert(err, IsNil)
	defer os.RemoveAll(tmpdir)
	start := time.Now().Round(0) // removes monotonic clocks value for deep equals
	startString := start.Format(time.RFC3339Nano)
	testdata := []struct {
		Content  string
		Expected []zeus.ClimateReport
		Error    string
	}{
		{
			Content: "# Starting date " + startString + `
# Time (ms) Relative Humidity (%) Temperature (°C)
0 50.0 21.23
502 51.3 24.5
`,
			Expected: []zeus.ClimateReport{
				zeus.ClimateReport{Time: start, Humidity: 50.0, Temperatures: []zeus.Temperature{21.23}},
				zeus.ClimateReport{Time: start.Add(502 * time.Millisecond), Humidity: 51.3, Temperatures: []zeus.Temperature{24.5}},
			},
		},
		{
			Content: "# Starting date " + startString + `
# Time (ms) Relative Humidity (%) Temperature (°C) Aux 1 (°C)
0 50.0 21.23 13.2
502 51.3 24.5 15.7
`,
			Expected: []zeus.ClimateReport{
				zeus.ClimateReport{Time: start, Humidity: 50.0, Temperatures: []zeus.Temperature{21.23, 13.2}},
				zeus.ClimateReport{Time: start.Add(502 * time.Millisecond), Humidity: 51.3, Temperatures: []zeus.Temperature{24.5, 15.7}},
			},
		},
		{
			Content: "# Starting date " + startString + `fo
# Time (ms) Relative Humidity (%) Temperature (°C) Aux 1 (°C)
0 50.0 21.23 13.2
502 51.3 24.5 15.7
`,
			Error: "parsing time \".*\": extra text: .*",
		},
		{
			Content: "# Starting date " + startString + `
# Time (ms) Relative Humidity (%)
0 50.0 21.23 13.2
502 51.3 24.5 15.7
`,
			Error: "invalid header .*",
		},
		{
			Content: "# Starting date " + startString + `
# Time (ms) Relative Humidity (%) Temperature (°C) Aux 1 (°C) Aux 2 (°C)
0 50.0 21.23 13.2 34.1
502 51.3 24.5 15.7
`,
			Error: "invalid line .*: too few values",
			Expected: []zeus.ClimateReport{
				zeus.ClimateReport{Time: start, Humidity: 50.0, Temperatures: []zeus.Temperature{21.23, 13.2, 34.1}},
			},
		},
	}
	filename := filepath.Join(tmpdir, "log.txt")
	for _, d := range testdata {
		err := ioutil.WriteFile(filename, []byte(d.Content), 0644)
		if c.Check(err, IsNil) == false {
			continue
		}
		result, err := ReadClimateFile(filename)
		if len(d.Error) > 0 {
			c.Check(err, ErrorMatches, d.Error)
		} else {
			c.Check(err, IsNil)
		}
		c.Check(result, DeepEquals, d.Expected)

	}
}

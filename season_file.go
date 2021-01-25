package zeus

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type SeasonFile struct {
	SlackUser string `yaml:"slack-user"`
	Zones     map[string]ZoneClimate
}

type deprecatedLine struct {
	name, comment string
	isError       bool
}

func checkDeprecatedLinesInZone(name string, items yaml.MapSlice) []deprecatedLine {
	var res []deprecatedLine = nil
	prefix := "zones." + name + "."
	for _, item := range items {
		switch item.Key.(string) {
		case "can-interface":
			res = append(res, deprecatedLine{name: prefix + "can-interface", comment: "value ignored", isError: false})
		case "devices-id":
			res = append(res, deprecatedLine{name: prefix + "devices-id", comment: "value ignored", isError: false})
		case "climate-report-file":
			res = append(res, deprecatedLine{
				name:    prefix + "climate-report-file",
				comment: "climate logs are saved under `/data/fort-user/fort-experiments/climate/" + name + ".<timestamp>.climate.txt`",
				isError: false,
			})
		}
	}

	return res
}

func checkDeprecatedLines(data []byte) ([]deprecatedLine, error) {
	parsed := yaml.MapSlice{}
	err := yaml.Unmarshal(data, &parsed)
	if err != nil {
		return nil, err
	}
	var res []deprecatedLine = nil

	for _, item := range parsed {
		key := item.Key.(string)
		if key == "emails" {
			res = append(res, deprecatedLine{name: key, comment: "value ignored", isError: false})
			continue
		}
		if key == "zones" {
			for _, zoneItem := range item.Value.(yaml.MapSlice) {
				zoneName := zoneItem.Key.(string)
				res = append(res, checkDeprecatedLinesInZone(zoneName, zoneItem.Value.(yaml.MapSlice))...)
			}
		}
	}

	return res, nil
}

func formatDeprecatedLines(lines []deprecatedLine, writer io.Writer) error {
	good := true
	for _, l := range lines {
		if l.isError == true {
			good = false
		}
	}
	if good == false {
		writer = bytes.NewBuffer(nil)
	}
	for _, l := range lines {
		prefix := "WARNING"
		suffix := " (will raise an error in a future release)"
		if l.isError == true {
			prefix = "ERROR"
			suffix = ""
		}
		fmt.Fprintf(writer, "%s: '%s' is deprecated%s: %s\n", prefix, l.name, suffix, l.comment)
	}
	if good == false {
		return fmt.Errorf("invalid season file:\n%s", writer.(*bytes.Buffer).Bytes())
	}
	return nil
}

func ReadSeasonFile(filename string, writer io.Writer) (*SeasonFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	s := &SeasonFile{}

	err = yaml.Unmarshal(data, s)
	if err != nil {
		return nil, err
	}

	lines, err := checkDeprecatedLines(data)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return s, nil
	}
	err = formatDeprecatedLines(lines, writer)
	if err != nil {
		return nil, err
	}

	return s, nil
}

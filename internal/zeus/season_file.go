package zeus

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

type SeasonFile struct {
	Zones map[string]ZoneClimate
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
		if key == "slack-user" {
			res = append(res, deprecatedLine{name: key, comment: "slack usage is deprecated, value is ignored", isError: false})
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

	var errs []error
	for _, l := range lines {
		if l.isError == true {
			errs = append(errs, fmt.Errorf("%s is deprecated: %s", l.name, l.comment))
		}
	}
	if len(errs) > 0 {
		return fmt.Errorf("invalid season file: %w", errors.Join(errs...))
	}

	log := logrus.New()
	log.SetOutput(writer)

	for _, l := range lines {
		log.WithFields(logrus.Fields{
			"field":   l.name,
			"comment": l.comment,
		}).Warn("deprecated field")
	}

	return nil
}

func ParseSeasonFile(content []byte) (*SeasonFile, error) {
	s := &SeasonFile{}

	err := yaml.Unmarshal(content, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func ReadSeasonFile(filename string, writer io.Writer) (*SeasonFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	s, err := ParseSeasonFile(data)
	if err != nil {
		return nil, err
	}

	lines, err := checkDeprecatedLines(data)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 || writer == nil {
		return s, nil
	}
	err = formatDeprecatedLines(lines, writer)
	if err != nil {
		return nil, err
	}

	return s, nil
}

func (f SeasonFile) WriteFile(filename string) error {
	data, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0644)
}

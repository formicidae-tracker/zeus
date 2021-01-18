package zeus

import (
	"io/ioutil"

	"gopkg.in/yaml.v2"
)

type SeasonFile struct {
	Zones map[string]ZoneClimate
}

func ReadSeasonFile(filename string) (*SeasonFile, error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	s := &SeasonFile{}

	err = yaml.Unmarshal(data, s)
	if err != nil {
		return nil, err
	}

	return s, nil
}

package main

import "fmt"

type Simulate struct {
}

var simOpts = Simulate{}

func (s *Simulate) Execute(args []string) error {

	// c := Config{}
	// f, err := os.Open(opts.Config)
	// if err != nil {
	// 	return err
	// }
	// defer f.Close()

	// buf, err := ioutil.ReadAll(f)
	// if err != nil {
	// 	return err
	// }
	// err = yaml.Unmarshal(buf, &c)
	// if err != nil {
	// 	return err
	// }

	return fmt.Errorf("Simulate not implemented")
}

func init() {
	parser.AddCommand("simulate", "simulate the climate change", "Simulate", &simOpts)

}

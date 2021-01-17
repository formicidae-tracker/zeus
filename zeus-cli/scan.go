package main

import "fmt"

type ScanCommand struct {
}

func (c *ScanCommand) Execute(args []string) error {
	nodes, err := Nodes()
	if err != nil {
		return err
	}
	format := " %15s | %s\n"
	fmt.Printf(format, "Node", "Status")
	fmt.Println("-----------------+--------------------------------------------------------------")

	for _, node := range nodes {
		running := false
		ignored := 0
		err := node.RunMethod("Zeus.Running", ignored, &running)
		if err != nil {
			return err
		}
		value := "Idle"
		if running == true {
			value = "Running"
		}
		fmt.Printf(format, node.Name, value)
	}
	return nil
}

func init() {
	_, err := parser.AddCommand("scan",
		"scan node on local network",
		"scans zeus node available on local network",
		&ScanCommand{})
	if err != nil {
		panic(err.Error())
	}

}

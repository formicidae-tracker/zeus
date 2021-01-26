package main

import (
	"fmt"

	"github.com/formicidae-tracker/zeus"
)

type ScanCommand struct {
}

func (c *ScanCommand) Execute(args []string) error {
	nodes, err := Nodes()
	if err != nil {
		return err
	}
	format := " %15s | %-7s | %-25s | %-10s \n"
	fmt.Printf(format, "Node", "Status", "Version", "Compatible")
	fmt.Println("-----------------+---------+---------------------------+------------")

	for _, node := range nodes {
		status := zeus.ZeusStatusReply{}
		ignored := 0
		err := node.RunMethod("Zeus.Status", ignored, &status)
		if err != nil {
			return err
		}
		statusValue := "Idle"
		if status.Running == true {
			statusValue = "Running"
		}
		compatibleValue := "✓"
		compatible, err := zeus.VersionAreCompatible(zeus.ZEUS_VERSION, status.Version)
		if err != nil || compatible == false {
			compatibleValue = "✗"
		}

		fmt.Printf(format, node.Name, statusValue, status.Version, compatibleValue)
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

package main

import (
	"fmt"
	"time"

	"github.com/formicidae-tracker/zeus"
)

type ScanCommand struct {
}

func (c *ScanCommand) Execute(args []string) error {
	now := time.Now()
	nodes, err := Nodes()
	if err != nil {
		return err
	}
	fmt.Println("┌──────────────────────┬─────────┬─────────────┬──────────────────────┬────────────┐")
	format := "│ %20s │ %-7s │ %-11s │ %-20s │ %-10s │\n"
	fmt.Printf(format, "Node", "Status", "Since", "Version", "Compatible")
	fmt.Println("├──────────────────────┼─────────┼─────────────┼──────────────────────┼────────────┤")

	for _, node := range nodes {
		status := zeus.ZeusStatusReply{}
		ignored := 0
		err := node.RunMethod("Zeus.Status", ignored, &status)
		if err != nil {
			fmt.Printf(format, node.Name, "n.a.", "n.a.", "<v0.3.0", "✗")
			continue
		}
		statusValue := "Idle"
		sinceValue := "n.a."
		if status.Running == true {
			statusValue = "Running"
			ellapsed := now.Sub(status.Since).Truncate(time.Second)
			sinceValue = ellapsed.String()
		}

		compatibleValue := "✓"
		compatible, err := zeus.VersionAreCompatible(zeus.ZEUS_VERSION, status.Version)
		if err != nil || compatible == false {
			compatibleValue = "✗"
		}

		fmt.Printf(format, node.Name, statusValue, sinceValue, status.Version, compatibleValue)
	}
	fmt.Println("└──────────────────────┴─────────┴─────────────┴──────────────────────┴────────────┘")

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

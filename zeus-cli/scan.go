package main

import (
	"log"
	"time"

	"github.com/atuleu/go-tablifier"
	"github.com/formicidae-tracker/zeus"
)

type ScanCommand struct {
}

type resultTableLine struct {
	Node       string
	Status     string
	Since      string
	Version    string
	Compatible string
}

func (c *ScanCommand) Execute(args []string) error {
	now := time.Now()
	nodes, err := Nodes()
	if err != nil {
		return err
	}
	lines := make([]resultTableLine, 0, len(nodes))

	for _, node := range nodes {
		status := zeus.ZeusStatusReply{}
		ignored := 0
		err := node.RunMethod("Zeus.Status", ignored, &status)
		line := resultTableLine{
			Node:       node.Name,
			Status:     "n.a.",
			Since:      "n.a.",
			Version:    "<0.3.0",
			Compatible: "✗",
		}
		if err != nil {
			log.Printf("Could not fetch %s status: %s", node.Name, err)
			lines = append(lines, line)
			continue
		}
		line.Status = "Idle"
		if status.Running == true {
			line.Status = "Running"
			ellapsed := now.Sub(status.Since).Truncate(time.Second)
			line.Since = ellapsed.String()
		}
		line.Version = status.Version
		compatible, err := zeus.VersionAreCompatible(zeus.ZEUS_VERSION, status.Version)
		if err == nil && compatible == true {
			line.Compatible = "✓"
		}
		lines = append(lines, line)
	}

	tablifier.Tablify(lines)

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

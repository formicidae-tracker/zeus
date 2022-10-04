package main

import (
	"fmt"
	"log"
	"sort"
	"time"

	"github.com/atuleu/go-tablifier"
	"github.com/formicidae-tracker/zeus"
)

type ScanCommand struct {
}

type resultTableLine struct {
	Zone       string
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
		status, err := node.Status()
		line := resultTableLine{
			Zone:       node.Name,
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
		line.Version = status.Version
		compatible, err := zeus.VersionAreCompatible(zeus.ZEUS_VERSION, status.Version)
		if err == nil && compatible == true {
			line.Compatible = "✓"
		}
		if status.Running == false {
			lines = append(lines, line)
			continue
		}
		ellapsed := now.Sub(status.Since.AsTime()).Truncate(time.Second)
		line.Since = ellapsed.String()
		if len(status.Zones) == 0 {
			line.Status = "Running"
			lines = append(lines, line)
			continue
		}

		for _, s := range status.Zones {
			line.Zone = node.Name + "." + s.Name
			line.Status = fmt.Sprintf("'%s' %.2f / %.2f °C %.2f / %.2f %% R.H.",
				s.Target.Name,
				s.Temperature,
				s.Target.Temperature,
				s.Humidity,
				s.Target.Humidity)

			lines = append(lines, line)
		}
	}

	sort.Slice(lines, func(i, j int) bool {
		iOn := lines[i].Status != "Idle" && lines[i].Status != "n.a."
		jOn := lines[j].Status != "Idle" && lines[j].Status != "n.a."
		if iOn == jOn {
			return lines[i].Zone < lines[j].Zone
		}
		return iOn
	})

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

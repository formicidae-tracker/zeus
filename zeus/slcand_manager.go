package main

import (
	"fmt"
	"os/exec"
)

type SlcandManager struct {
	ifname, devname string
	os.Exec("Command"
}

func (m *SlcandManager) open() (err error) {
	cmd := exec.Command("slcand", "-ofs", "5", "-S", "115200", m.devname, m.ifname)

	return fmt.Errorf("Not Yet Implemented")
}

func OpenSlcand(ifname, devname string) (*SlcandManager, error) {
	return nil, fmt.Errorf("Not yet implemented")
}

func (*SlcandManager) Close() error {
	return fmt.Errorf("Not yet implemented")
}

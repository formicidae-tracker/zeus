package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"syscall"
	"time"
)

type slcandLogger log.Logger

func (l *slcandLogger) Write(data []byte) (int, error) {
	(*log.Logger)(l).Printf("[slcand] %s", string(data))
	return len(data), nil
}

type SlcandManager struct {
	ifname   string
	devname  string
	logger   *slcandLogger
	cmd      *exec.Cmd
	cmdError chan error
}

func (m *SlcandManager) open() (err error) {
	m.cmd = exec.Command("slcand", "-ofs", "5", "-S", "115200", "-F", m.devname, m.ifname)
	m.cmd.Stdout = m.logger
	m.cmd.Stderr = m.logger

	go func() {
		m.cmdError <- m.cmd.Run()
		close(m.cmdError)
	}()

	select {
	case err := <-m.cmdError:
		return fmt.Errorf("Could not open slcand: %s", err)
	case <-time.After(1500 * time.Millisecond):
	}

	ipCmd := exec.Command("ip", "link", "set", m.ifname, "up")
	out, err := ipCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not set %s up: %s", m.ifname, string(out))
	}

	return nil
}

func Open(ifname, devname string) (*SlcandManager, error) {
	m := &SlcandManager{
		ifname:   ifname,
		devname:  devname,
		logger:   (*slcandLogger)(log.New(os.Stderr, fmt.Sprintf("[slcand/%s]", ifname), 0)),
		cmdError: make(chan error),
	}
	if err := m.open(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *SlcandManager) Close() error {
	ipCmd := exec.Command("ip", "link", "set", m.ifname, "down")
	out, err := ipCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not set %s down: %s", m.ifname, string(out))
	}
	m.cmd.Process.Signal(syscall.SIGINT)
	return <-m.cmdError
}

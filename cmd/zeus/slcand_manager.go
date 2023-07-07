package main

import (
	"bufio"
	"fmt"
	"io"
	"os/exec"
	"path"
	"syscall"
	"time"

	"github.com/formicidae-tracker/olympus/pkg/tm"
	"github.com/sirupsen/logrus"
)

type SlcandManager struct {
	ifname   string
	devname  string
	logger   *logrus.Entry
	cmd      *exec.Cmd
	cmdError chan error
}

func (m *SlcandManager) open() (err error) {
	m.cmd = exec.Command("slcand", "-ofs", "5", "-S", "115200", "-F", m.devname, m.ifname)
	//avoids the daemon to get the signal from terminal, we take care to do it ourselves
	m.cmd.SysProcAttr = &syscall.SysProcAttr{
		Setpgid: true,
		Pgid:    0,
	}

	stdout, err := m.cmd.StdoutPipe()
	if err != nil {
		return err
	}
	stderr, err := m.cmd.StderrPipe()
	if err != nil {
		return err
	}

	scanner := bufio.NewScanner(io.MultiReader(stdout, stderr))

	go func() {
		err := m.cmd.Start()
		if err != nil {
			m.cmdError <- err
			close(m.cmdError)
			return
		}

		for scanner.Scan() {
			m.logger.WithField("output", scanner.Text()).Info("slcand output")
		}

		m.cmdError <- m.cmd.Wait()
		close(m.cmdError)
	}()

	select {
	case err := <-m.cmdError:
		return fmt.Errorf("Could not open slcand: %s", err)
	case <-time.After(500 * time.Millisecond):
	}
	m.logger.WithField("interface", m.ifname).Info("set interface link up")
	ipCmd := exec.Command("ip", "link", "set", m.ifname, "up")
	out, err := ipCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not set %s up: %s", m.ifname, string(out))
	}

	return nil
}

func OpenSlcand(ifname, devname string) (*SlcandManager, error) {
	m := &SlcandManager{
		ifname:   ifname,
		devname:  devname,
		logger:   tm.NewLogger(path.Join("slcand", ifname)),
		cmdError: make(chan error),
	}
	if err := m.open(); err != nil {
		return nil, err
	}
	return m, nil
}

func (m *SlcandManager) Close() error {
	m.logger.WithField("interface", m.ifname).Info("set interface link down")
	ipCmd := exec.Command("ip", "link", "set", m.ifname, "down")
	out, err := ipCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("Could not set %s down: %s", m.ifname, string(out))
	}
	m.cmd.Process.Signal(syscall.SIGINT)
	return <-m.cmdError
}

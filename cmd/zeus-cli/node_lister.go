package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/adrg/xdg"
	"github.com/grandcat/zeroconf"
	"gopkg.in/yaml.v2"
)

const CACHE_TTL = 5 * time.Second

type NodeLister struct {
	CacheDate time.Time       `yaml:"date"`
	Cache     map[string]Node `yaml:"nodes"`
}

func NewLister() *NodeLister {
	l := &NodeLister{}
	l.load()
	return l
}

func (l *NodeLister) ListNodes() (map[string]Node, error) {
	if time.Now().Before(l.CacheDate.Add(CACHE_TTL)) == true {
		return l.Cache, nil
	}
	return l.listNodes()
}

func (l *NodeLister) cacheFilePath() string {
	return filepath.Join(xdg.CacheHome, "fort/zeus/node.cache")
}

func (l *NodeLister) load() {
	depreceatedDate := time.Now().Add(-2 * CACHE_TTL)
	l.CacheDate = depreceatedDate
	content, err := ioutil.ReadFile(l.cacheFilePath())
	if err != nil {
		return
	}
	err = yaml.Unmarshal(content, l)
	if err != nil {
		l.CacheDate = depreceatedDate
	}
}

func (l *NodeLister) save() error {
	if err := os.MkdirAll(filepath.Dir(l.cacheFilePath()), 0755); err != nil {
		return err
	}
	content, err := yaml.Marshal(l)
	if err != nil {
		return nil
	}
	return ioutil.WriteFile(l.cacheFilePath(), content, 0644)
}

func (l *NodeLister) listNodes() (nodes map[string]Node, err error) {
	defer func() {
		if err != nil {
			return
		}
		l.Cache = nodes
		l.CacheDate = time.Now()
		l.save()
	}()
	nodes = nil

	var resolver *zeroconf.Resolver
	resolver, err = zeroconf.NewResolver(nil)
	if err != nil {
		return
	}
	entries := make(chan *zeroconf.ServiceEntry, 100)
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	err = resolver.Browse(ctx, "_zeus._tcp", "local.", entries)
	if err != nil {
		err = fmt.Errorf("Could not browse for zeus instances: %s", err)
		return
	}

	<-ctx.Done()

	nodes = make(map[string]Node)

	for e := range entries {
		name := strings.TrimPrefix(e.Instance, "zeus.")
		address := strings.TrimSuffix(e.HostName, ".")
		port := e.Port
		nodes[name] = Node{Name: name, Address: address, Port: port}
	}
	err = nil
	return
}

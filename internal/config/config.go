package config

import (
	"fmt"
	"os"
	"sort"
	"sync"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Servers []ServerConfig `yaml:"servers"`
	Tokens  []TokenConfig  `yaml:"tokens"`

	once            sync.Once                `yaml:"-"`
	serverByPort    map[int]ServerConfig     `yaml:"-"`
	serverByName    map[string]ServerConfig  `yaml:"-"`
	tokenByID       map[int32]TokenConfig    `yaml:"-"`
	replicasByToken map[int32][]ServerConfig `yaml:"-"`
}

type ServerConfig struct {
	Name        string `yaml:"name"`
	Host        string `yaml:"host"`
	Port        int    `yaml:"port"`
	SleepMs     int    `yaml:"sleep_ms"`
	NegativeAck bool   `yaml:"negative_ack"`
}

type TokenConfig struct {
	ID       int32    `yaml:"id"`
	Readers  []string `yaml:"readers"`
	Writers  []string `yaml:"writers"`
	Replicas []string `yaml:"replicas"`
}

func Load(path string) (*Config, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var c Config
	if err := yaml.Unmarshal(b, &c); err != nil {
		return nil, fmt.Errorf("parse config yaml: %w", err)
	}

	if len(c.Servers) == 0 {
		return nil, fmt.Errorf("config has no servers")
	}

	for i := range c.Servers {
		if c.Servers[i].Name == "" {
			c.Servers[i].Name = fmt.Sprintf("s%d", i+1)
		}
	}

	for i := range c.Tokens {
		if len(c.Tokens[i].Replicas) == 0 {
			c.Tokens[i].Replicas = dedupeStrings(append(append([]string{}, c.Tokens[i].Readers...), c.Tokens[i].Writers...))
		}
	}

	c.ensureIndexes()

	return &c, nil
}

func (c *Config) ensureIndexes() {
	c.once.Do(func() {
		c.serverByPort = make(map[int]ServerConfig, len(c.Servers))
		c.serverByName = make(map[string]ServerConfig, len(c.Servers))
		for _, s := range c.Servers {
			c.serverByPort[s.Port] = s
			c.serverByName[s.Name] = s
		}

		c.tokenByID = make(map[int32]TokenConfig, len(c.Tokens))
		c.replicasByToken = make(map[int32][]ServerConfig, len(c.Tokens))
		for _, t := range c.Tokens {
			c.tokenByID[t.ID] = t
			replicas := make([]ServerConfig, 0, len(t.Replicas))
			for _, name := range dedupeStrings(t.Replicas) {
				s, ok := c.serverByName[name]
				if ok {
					replicas = append(replicas, s)
				}
			}
			sort.Slice(replicas, func(i, j int) bool {
				return replicas[i].Port < replicas[j].Port
			})
			c.replicasByToken[t.ID] = replicas
		}
	})
}

func (c *Config) ServerByPort(port int) (ServerConfig, bool) {
	c.ensureIndexes()
	s, ok := c.serverByPort[port]
	return s, ok
}

func (c *Config) ServerByName(name string) (ServerConfig, bool) {
	c.ensureIndexes()
	s, ok := c.serverByName[name]
	return s, ok
}

func (c *Config) TokenByID(id int32) (TokenConfig, bool) {
	c.ensureIndexes()
	t, ok := c.tokenByID[id]
	return t, ok
}

func (c *Config) IsReader(server string, tokenID int32) bool {
	t, ok := c.TokenByID(tokenID)
	if !ok {
		return false
	}
	for _, n := range t.Readers {
		if n == server {
			return true
		}
	}
	return false
}

func (c *Config) IsWriter(server string, tokenID int32) bool {
	t, ok := c.TokenByID(tokenID)
	if !ok {
		return false
	}
	for _, n := range t.Writers {
		if n == server {
			return true
		}
	}
	return false
}

func (c *Config) ReplicaServers(tokenID int32) []ServerConfig {
	c.ensureIndexes()
	replicas, ok := c.replicasByToken[tokenID]
	if !ok {
		return nil
	}
	out := make([]ServerConfig, len(replicas))
	copy(out, replicas)
	return out
}

func dedupeStrings(in []string) []string {
	seen := make(map[string]struct{}, len(in))
	out := make([]string, 0, len(in))
	for _, v := range in {
		if v == "" {
			continue
		}
		if _, ok := seen[v]; ok {
			continue
		}
		seen[v] = struct{}{}
		out = append(out, v)
	}
	return out
}

package config

import (
	"fmt"
	"os"
	"sort"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Servers []ServerConfig `yaml:"servers"`
	Tokens  []TokenConfig  `yaml:"tokens"`
}

type ServerConfig struct {
	Name      string `yaml:"name"`
	Host      string `yaml:"host"`
	Port      int    `yaml:"port"`
	SleepMs   int    `yaml:"sleep_ms"`
	NegativeAck bool `yaml:"negative_ack"`
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

	return &c, nil
}

func (c *Config) ServerByPort(port int) (ServerConfig, bool) {
	for _, s := range c.Servers {
		if s.Port == port {
			return s, true
		}
	}
	return ServerConfig{}, false
}

func (c *Config) ServerByName(name string) (ServerConfig, bool) {
	for _, s := range c.Servers {
		if s.Name == name {
			return s, true
		}
	}
	return ServerConfig{}, false
}

func (c *Config) TokenByID(id int32) (TokenConfig, bool) {
	for _, t := range c.Tokens {
		if t.ID == id {
			return t, true
		}
	}
	return TokenConfig{}, false
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
	t, ok := c.TokenByID(tokenID)
	if !ok {
		return nil
	}
	out := make([]ServerConfig, 0, len(t.Replicas))
	for _, name := range dedupeStrings(t.Replicas) {
		s, ok := c.ServerByName(name)
		if ok {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Port < out[j].Port
	})
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

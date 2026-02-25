package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLoadAppliesDefaultsAndIndexes(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "token_config.yml")
	content := `
servers:
  - host: 127.0.0.1
    port: 5001
  - name: s2
    host: 127.0.0.1
    port: 5002
tokens:
  - id: 1
    readers: [s1, s1, s2]
    writers: [s2]
`
	if err := os.WriteFile(cfgPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := Load(cfgPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}

	s1, ok := cfg.ServerByName("s1")
	if !ok {
		t.Fatalf("expected default server name s1")
	}
	if s1.Port != 5001 {
		t.Fatalf("unexpected s1 port: got %d", s1.Port)
	}

	if _, ok := cfg.ServerByPort(5002); !ok {
		t.Fatalf("expected to find server by port")
	}

	tk, ok := cfg.TokenByID(1)
	if !ok {
		t.Fatalf("expected token id 1")
	}
	if len(tk.Replicas) == 0 {
		t.Fatalf("expected fallback replicas from readers/writers")
	}

	if !cfg.IsReader("s2", 1) {
		t.Fatalf("expected s2 reader access")
	}
	if cfg.IsWriter("s1", 1) {
		t.Fatalf("did not expect s1 writer access")
	}

	replicas := cfg.ReplicaServers(1)
	if len(replicas) != 2 {
		t.Fatalf("expected 2 deduped replicas, got %d", len(replicas))
	}
	if replicas[0].Port != 5001 || replicas[1].Port != 5002 {
		t.Fatalf("expected replicas sorted by port, got %+v", replicas)
	}
}

func TestReplicaServersDedupesAndFiltersUnknown(t *testing.T) {
	t.Parallel()

	cfg := &Config{
		Servers: []ServerConfig{
			{Name: "s1", Port: 5003},
			{Name: "s2", Port: 5001},
			{Name: "s3", Port: 5002},
		},
		Tokens: []TokenConfig{
			{ID: 7, Replicas: []string{"s3", "s2", "missing", "s2", "s1"}},
		},
	}

	replicas := cfg.ReplicaServers(7)
	if len(replicas) != 3 {
		t.Fatalf("expected 3 known replicas, got %d", len(replicas))
	}

	gotPorts := []int{replicas[0].Port, replicas[1].Port, replicas[2].Port}
	wantPorts := []int{5001, 5002, 5003}
	if !reflect.DeepEqual(gotPorts, wantPorts) {
		t.Fatalf("unexpected sorted replica ports: got %v want %v", gotPorts, wantPorts)
	}
}

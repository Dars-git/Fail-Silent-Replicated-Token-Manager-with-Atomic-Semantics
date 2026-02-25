package config

import "testing"

func benchmarkConfig() *Config {
	servers := make([]ServerConfig, 0, 100)
	for i := 0; i < 100; i++ {
		servers = append(servers, ServerConfig{
			Name: "s" + string(rune('a'+(i%26))) + string(rune('A'+(i/26))),
			Port: 5000 + i,
		})
	}
	// Stable names used by tokens.
	servers[0].Name = "s1"
	servers[1].Name = "s2"
	servers[2].Name = "s3"

	tokens := make([]TokenConfig, 0, 1000)
	for i := 0; i < 1000; i++ {
		tokens = append(tokens, TokenConfig{
			ID:       int32(i + 1),
			Readers:  []string{"s1", "s2", "s3"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1", "s2", "s3"},
		})
	}
	cfg := &Config{
		Servers: servers,
		Tokens:  tokens,
	}
	cfg.ensureIndexes()
	return cfg
}

func BenchmarkServerByName(b *testing.B) {
	cfg := benchmarkConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if _, ok := cfg.ServerByName("s3"); !ok {
			b.Fatal("server not found")
		}
	}
}

func BenchmarkTokenByID(b *testing.B) {
	cfg := benchmarkConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := int32((i % 1000) + 1)
		if _, ok := cfg.TokenByID(id); !ok {
			b.Fatal("token not found")
		}
	}
}

func BenchmarkReplicaServers(b *testing.B) {
	cfg := benchmarkConfig()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		id := int32((i % 1000) + 1)
		if len(cfg.ReplicaServers(id)) != 3 {
			b.Fatal("unexpected replicas")
		}
	}
}

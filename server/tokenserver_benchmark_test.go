package main

import (
	"context"
	"testing"

	"tokenmanager/internal/config"
	"tokenmanager/internal/tokenstore"
	pb "tokenmanager/token_management"
)

func benchmarkServer() *TokenManagerServer {
	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{
			{ID: 1, Readers: []string{"s1"}, Writers: []string{"s1"}, Replicas: []string{"s1"}},
		},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.store.Set(1, tokenstore.Value{
		WTS:        "00000000000000000001-s1-000001",
		PartialVal: 10,
		Name:       "x",
		Low:        1,
		Mid:        2,
		High:       7,
	})
	return s
}

func BenchmarkWriteTokenSelfReplica(b *testing.B) {
	s := benchmarkServer()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.WriteToken(ctx, &pb.WriteTokenMsg{
			TokenId:      1,
			PartialValue: uint64(i + 1),
			Name:         "bench",
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkReadTokenSelfReplica(b *testing.B) {
	s := benchmarkServer()
	ctx := context.Background()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := s.ReadToken(ctx, &pb.Token{TokenId: 1})
		if err != nil {
			b.Fatal(err)
		}
	}
}

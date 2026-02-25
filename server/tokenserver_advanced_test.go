package main

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"tokenmanager/internal/config"
	"tokenmanager/internal/tokenstore"
	pb "tokenmanager/token_management"
)

type mockTransport struct {
	mu         sync.Mutex
	writeCalls int
	readCalls  int
	writeFn    func(ctx context.Context, peer config.ServerConfig, req *pb.WriteBroadcastRequest) (*pb.WriteBroadcastResponse, error)
	readFn     func(ctx context.Context, peer config.ServerConfig, tokenID int32) (*pb.ReadBroadcastResponse, error)
}

func (m *mockTransport) WriteBroadcast(ctx context.Context, peer config.ServerConfig, req *pb.WriteBroadcastRequest) (*pb.WriteBroadcastResponse, error) {
	m.mu.Lock()
	m.writeCalls++
	fn := m.writeFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, peer, req)
	}
	return &pb.WriteBroadcastResponse{Ack: 1}, nil
}

func (m *mockTransport) ReadBroadcast(ctx context.Context, peer config.ServerConfig, tokenID int32) (*pb.ReadBroadcastResponse, error) {
	m.mu.Lock()
	m.readCalls++
	fn := m.readFn
	m.mu.Unlock()
	if fn != nil {
		return fn(ctx, peer, tokenID)
	}
	return &pb.ReadBroadcastResponse{}, nil
}

func TestWriteTokenUsesTransportAbstraction(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Port: 50051},
			{Name: "s2", Port: 50052},
			{Name: "s3", Port: 50053},
		},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1", "s2", "s3"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1", "s2", "s3"},
		}},
	}
	mt := &mockTransport{}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.transport = mt

	resp, err := s.WriteToken(context.Background(), &pb.WriteTokenMsg{
		TokenId: 1,
		Name:    "abc",
		Low:     1,
		Mid:     2,
		High:    3,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 1 {
		t.Fatalf("expected write ack=1, got %+v", resp)
	}
	if mt.writeCalls != 2 {
		t.Fatalf("expected 2 outbound write broadcasts, got %d", mt.writeCalls)
	}
}

func TestWriteTokenContextCancellationPropagatesToBroadcasts(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Port: 50051},
			{Name: "s2", Port: 50052},
			{Name: "s3", Port: 50053},
		},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1", "s2", "s3"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1", "s2", "s3"},
		}},
	}

	mt := &mockTransport{
		writeFn: func(ctx context.Context, _ config.ServerConfig, _ *pb.WriteBroadcastRequest) (*pb.WriteBroadcastResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.transport = mt
	s.requestTimeout = 500 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err := s.WriteToken(ctx, &pb.WriteTokenMsg{TokenId: 1, PartialValue: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "failed quorum") {
		t.Fatalf("expected quorum failure after context cancel, got %+v", resp)
	}
}

func TestReadTokenContextCancellationPropagatesToBroadcasts(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Port: 50051},
			{Name: "s2", Port: 50052},
			{Name: "s3", Port: 50053},
		},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1", "s2", "s3"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1", "s2", "s3"},
		}},
	}

	mt := &mockTransport{
		readFn: func(ctx context.Context, _ config.ServerConfig, _ int32) (*pb.ReadBroadcastResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
		writeFn: func(ctx context.Context, _ config.ServerConfig, _ *pb.WriteBroadcastRequest) (*pb.WriteBroadcastResponse, error) {
			<-ctx.Done()
			return nil, ctx.Err()
		},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.transport = mt
	s.requestTimeout = 500 * time.Millisecond
	s.store.Set(1, tokenstore.Value{WTS: "9-s1-000001", PartialVal: 9})

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	resp, err := s.ReadToken(ctx, &pb.Token{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "quorum failed") {
		t.Fatalf("expected write-back quorum failure after context cancel, got %+v", resp)
	}
}

func TestDeterministicWTSHooks(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	s.nowUnixNano = func() int64 { return 1234567890 }
	seq := uint64(0)
	s.nextSequence = func() uint64 {
		seq++
		return seq
	}

	w1 := s.nextWTS()
	w2 := s.nextWTS()
	if w1 != "00000000001234567890-s1-000001" {
		t.Fatalf("unexpected first wts: %s", w1)
	}
	if w2 != "00000000001234567890-s1-000002" {
		t.Fatalf("unexpected second wts: %s", w2)
	}
}

func TestQuorumMathTable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		total  int
		acks   int
		quorum int
		hasQrm bool
	}{
		{name: "zero", total: 0, acks: 0, quorum: 0, hasQrm: true},
		{name: "one-pass", total: 1, acks: 1, quorum: 1, hasQrm: true},
		{name: "two-fail", total: 2, acks: 1, quorum: 2, hasQrm: false},
		{name: "three-pass", total: 3, acks: 2, quorum: 2, hasQrm: true},
		{name: "five-fail", total: 5, acks: 2, quorum: 3, hasQrm: false},
		{name: "five-pass", total: 5, acks: 3, quorum: 3, hasQrm: true},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if got := requiredQuorum(tc.total); got != tc.quorum {
				t.Fatalf("requiredQuorum(%d)=%d want=%d", tc.total, got, tc.quorum)
			}
			if got := hasQuorum(tc.acks, tc.total); got != tc.hasQrm {
				t.Fatalf("hasQuorum(%d,%d)=%v want=%v", tc.acks, tc.total, got, tc.hasQrm)
			}
		})
	}
}

func TestMetricsSnapshotCountersAndLatency(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	if _, err := s.WriteToken(context.Background(), &pb.WriteTokenMsg{TokenId: 1, PartialValue: 5}); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if _, err := s.ReadToken(context.Background(), &pb.Token{TokenId: 1}); err != nil {
		t.Fatalf("read token: %v", err)
	}

	snap := s.MetricsSnapshot()
	if snap.WriteTokenCalls != 1 || snap.ReadTokenCalls != 1 {
		t.Fatalf("unexpected call counters: %+v", snap)
	}
	if snap.WriteTokenLatencyAvgMicros < 0 || snap.ReadTokenLatencyAvgMicros < 0 {
		t.Fatalf("latency averages should not be negative: %+v", snap)
	}
}

func TestReadBroadcastOutboundErrorIncrementsMetric(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Port: 50051},
			{Name: "s2", Port: 50052},
		},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1", "s2"},
			Writers:  []string{"s1"},
			Replicas: []string{"s1", "s2"},
		}},
	}
	mt := &mockTransport{
		readFn: func(_ context.Context, _ config.ServerConfig, _ int32) (*pb.ReadBroadcastResponse, error) {
			return nil, errors.New("read fail")
		},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.transport = mt
	s.store.Set(1, tokenstore.Value{WTS: "10-s1-000001", PartialVal: 10})

	if _, err := s.ReadToken(context.Background(), &pb.Token{TokenId: 1}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	snap := s.MetricsSnapshot()
	if snap.OutboundReadBroadcastErrors == 0 {
		t.Fatalf("expected outbound read broadcast error metric to increment")
	}
}

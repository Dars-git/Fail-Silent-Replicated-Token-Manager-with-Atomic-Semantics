package main

import (
	"context"
	"strings"
	"testing"
	"time"

	"tokenmanager/internal/config"
	"tokenmanager/internal/tokenstore"
	pb "tokenmanager/token_management"
)

func newTestServer(cfg *config.Config, self config.ServerConfig) *TokenManagerServer {
	return newTokenManagerServer(cfg, tokenstore.New(), self, 100*time.Millisecond)
}

func TestWriteTokenNilRequest(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	resp, err := s.WriteToken(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error")
	}
}

func TestWriteTokenAuthorizationFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1"},
			Writers:  []string{"s2"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.WriteToken(context.Background(), &pb.WriteTokenMsg{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "authorization failed") {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteTokenNoReplicas(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Writers:  []string{"s1"},
			Replicas: []string{},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.WriteToken(context.Background(), &pb.WriteTokenMsg{TokenId: 1, PartialValue: 7})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "no replica nodes found") {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestWriteTokenSelfReplicaSuccessAndDerivedPartial(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Writers:  []string{"s1"},
			Readers:  []string{"s1"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.WriteToken(context.Background(), &pb.WriteTokenMsg{
		TokenId: 1,
		Name:    "abc",
		Low:     2,
		Mid:     3,
		High:    5,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 1 {
		t.Fatalf("expected ack=1, got %+v", resp)
	}
	if resp.FinalVal != 10 {
		t.Fatalf("expected derived partial=10, got %d", resp.FinalVal)
	}
	if resp.Wts == "" {
		t.Fatalf("expected non-empty wts")
	}

	v, ok := s.store.Get(1)
	if !ok {
		t.Fatalf("expected token stored")
	}
	if v.PartialVal != 10 || v.WTS == "" {
		t.Fatalf("unexpected stored value: %+v", v)
	}
}

func TestReadTokenNilRequest(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	resp, err := s.ReadToken(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error")
	}
}

func TestReadTokenAuthorizationFailure(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s2"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.ReadToken(context.Background(), &pb.Token{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "authorization failed") {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestReadTokenNoValueYet(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.ReadToken(context.Background(), &pb.Token{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 || !strings.Contains(resp.Message, "no value yet") {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestReadTokenReturnsLocalValue(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens: []config.TokenConfig{{
			ID:       1,
			Readers:  []string{"s1"},
			Replicas: []string{"s1"},
		}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.store.Set(1, tokenstore.Value{
		WTS:        "0001-s1-000001",
		PartialVal: 130,
		Name:       "abc",
		Low:        5,
		Mid:        25,
		High:       100,
	})

	resp, err := s.ReadToken(context.Background(), &pb.Token{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 1 {
		t.Fatalf("expected ack=1, got %+v", resp)
	}
	if resp.Wts != "0001-s1-000001" || resp.FinalVal != 130 {
		t.Fatalf("unexpected read response: %+v", resp)
	}
}

func TestWriteBroadcastNilRequest(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	resp, err := s.WriteBroadcast(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error")
	}
}

func TestWriteBroadcastRejectsUnknownToken(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})

	resp, err := s.WriteBroadcast(context.Background(), &pb.WriteBroadcastRequest{
		TokenId: 99,
		Wts:     "1",
		HashVal: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 0 {
		t.Fatalf("expected ack=0 for unknown token, got %+v", resp)
	}
}

func TestWriteBroadcastNegativeAckOnlyForWrite(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens:  []config.TokenConfig{{ID: 1}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051, NegativeAck: true})

	writeResp, err := s.WriteBroadcast(context.Background(), &pb.WriteBroadcastRequest{
		TokenId:     1,
		Wts:         "1",
		HashVal:     10,
		ReadingFlag: false,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if writeResp.Ack != 0 {
		t.Fatalf("expected ack=0 for negative ack mode, got %+v", writeResp)
	}

	readBackResp, err := s.WriteBroadcast(context.Background(), &pb.WriteBroadcastRequest{
		TokenId:     1,
		Wts:         "1",
		HashVal:     10,
		ReadingFlag: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if readBackResp.Ack != 1 {
		t.Fatalf("expected ack=1 for read-impose write-back, got %+v", readBackResp)
	}
}

func TestWriteBroadcastStaleTimestampStillAcks(t *testing.T) {
	t.Parallel()

	cfg := &config.Config{
		Servers: []config.ServerConfig{{Name: "s1", Port: 50051}},
		Tokens:  []config.TokenConfig{{ID: 1}},
	}
	s := newTestServer(cfg, config.ServerConfig{Name: "s1", Port: 50051})
	s.store.Set(1, tokenstore.Value{WTS: "2", PartialVal: 20})

	resp, err := s.WriteBroadcast(context.Background(), &pb.WriteBroadcastRequest{
		TokenId: 1,
		Wts:     "1",
		HashVal: 10,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Ack != 1 {
		t.Fatalf("expected ack=1 for stale no-op, got %+v", resp)
	}

	v, _ := s.store.Get(1)
	if v.WTS != "2" || v.PartialVal != 20 {
		t.Fatalf("stale write should not override newer value: %+v", v)
	}
}

func TestReadBroadcastNilRequest(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	resp, err := s.ReadBroadcast(context.Background(), nil)
	if err == nil {
		t.Fatalf("expected error for nil request")
	}
	if resp != nil {
		t.Fatalf("expected nil response on error")
	}
}

func TestReadBroadcastMissingValue(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	resp, err := s.ReadBroadcast(context.Background(), &pb.ReadBroadcastRequest{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Wts != "" || resp.FinalVal != 0 {
		t.Fatalf("expected empty response for unknown token value, got %+v", resp)
	}
}

func TestReadBroadcastReturnsStoredValue(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	s.store.Set(1, tokenstore.Value{
		WTS:        "100-s1-000001",
		PartialVal: 88,
		Name:       "x",
		Low:        1,
		Mid:        2,
		High:       3,
	})

	resp, err := s.ReadBroadcast(context.Background(), &pb.ReadBroadcastRequest{TokenId: 1})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Wts != "100-s1-000001" || resp.FinalVal != 88 {
		t.Fatalf("unexpected response: %+v", resp)
	}
}

func TestNextWTSProducesUniqueValues(t *testing.T) {
	t.Parallel()

	s := newTestServer(&config.Config{}, config.ServerConfig{Name: "s1"})
	w1 := s.nextWTS()
	w2 := s.nextWTS()

	if w1 == w2 {
		t.Fatalf("expected unique timestamps, got %q and %q", w1, w2)
	}
	if !strings.Contains(w1, "-s1-") || !strings.Contains(w2, "-s1-") {
		t.Fatalf("expected server name in wts values: %q %q", w1, w2)
	}
}

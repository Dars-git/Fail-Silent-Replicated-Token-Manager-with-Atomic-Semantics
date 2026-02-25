package main

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"tokenmanager/internal/config"
	"tokenmanager/internal/tokenstore"
	pb "tokenmanager/token_management"
)

func freePort(t *testing.T) int {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen free port: %v", err)
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port
}

func dialClient(t *testing.T, host string, port int) (*grpc.ClientConn, pb.TokenManagerClient) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	conn, err := grpc.DialContext(
		ctx,
		fmt.Sprintf("%s:%d", host, port),
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
		grpc.WithBlock(),
	)
	if err != nil {
		t.Fatalf("dial %s:%d: %v", host, port, err)
	}
	return conn, pb.NewTokenManagerClient(conn)
}

func startCluster(t *testing.T, cfg *config.Config) func() {
	t.Helper()

	type running struct {
		lis net.Listener
		srv *grpc.Server
	}
	instances := make([]running, 0, len(cfg.Servers))
	for _, self := range cfg.Servers {
		lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", self.Host, self.Port))
		if err != nil {
			t.Fatalf("listen %s:%d: %v", self.Host, self.Port, err)
		}
		grpcServer := grpc.NewServer()

		store := tokenstore.New()
		for _, tk := range cfg.Tokens {
			for _, replica := range tk.Replicas {
				if replica == self.Name {
					store.Set(tk.ID, tokenstore.Value{})
					break
				}
			}
		}
		pb.RegisterTokenManagerServer(grpcServer, newTokenManagerServer(cfg, store, self, 2*time.Second))
		go func(gs *grpc.Server, ln net.Listener) {
			_ = gs.Serve(ln)
		}(grpcServer, lis)
		instances = append(instances, running{lis: lis, srv: grpcServer})
	}

	time.Sleep(120 * time.Millisecond)
	return func() {
		for _, in := range instances {
			in.srv.Stop()
			_ = in.lis.Close()
		}
	}
}

func TestIntegrationThreeNodeWriteRead(t *testing.T) {
	t.Parallel()

	p1 := freePort(t)
	p2 := freePort(t)
	p3 := freePort(t)

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Host: "127.0.0.1", Port: p1},
			{Name: "s2", Host: "127.0.0.1", Port: p2},
			{Name: "s3", Host: "127.0.0.1", Port: p3},
		},
		Tokens: []config.TokenConfig{
			{ID: 1, Readers: []string{"s1", "s2", "s3"}, Writers: []string{"s1"}, Replicas: []string{"s1", "s2", "s3"}},
		},
	}
	stop := startCluster(t, cfg)
	defer stop()

	conn1, cli1 := dialClient(t, "127.0.0.1", p1)
	defer conn1.Close()
	conn2, cli2 := dialClient(t, "127.0.0.1", p2)
	defer conn2.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	wr, err := cli1.WriteToken(ctx, &pb.WriteTokenMsg{
		TokenId: 1,
		Name:    "abc",
		Low:     5,
		Mid:     25,
		High:    100,
	})
	if err != nil {
		t.Fatalf("write rpc failed: %v", err)
	}
	if wr.Ack != 1 {
		t.Fatalf("expected write ack=1, got %+v", wr)
	}

	rd, err := cli2.ReadToken(ctx, &pb.Token{TokenId: 1})
	if err != nil {
		t.Fatalf("read rpc failed: %v", err)
	}
	if rd.Ack != 1 || rd.FinalVal != 130 || rd.Wts == "" {
		t.Fatalf("unexpected read response: %+v", rd)
	}
}

func TestIntegrationThreeNodeNegativeAckStillQuorum(t *testing.T) {
	t.Parallel()

	p1 := freePort(t)
	p2 := freePort(t)
	p3 := freePort(t)

	cfg := &config.Config{
		Servers: []config.ServerConfig{
			{Name: "s1", Host: "127.0.0.1", Port: p1},
			{Name: "s2", Host: "127.0.0.1", Port: p2, NegativeAck: true},
			{Name: "s3", Host: "127.0.0.1", Port: p3},
		},
		Tokens: []config.TokenConfig{
			{ID: 1, Readers: []string{"s1", "s2", "s3"}, Writers: []string{"s1"}, Replicas: []string{"s1", "s2", "s3"}},
		},
	}
	stop := startCluster(t, cfg)
	defer stop()

	conn1, cli1 := dialClient(t, "127.0.0.1", p1)
	defer conn1.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	wr, err := cli1.WriteToken(ctx, &pb.WriteTokenMsg{TokenId: 1, PartialValue: 11})
	if err != nil {
		t.Fatalf("write rpc failed: %v", err)
	}
	if wr.Ack != 1 {
		t.Fatalf("expected write ack=1 with one negative ack replica, got %+v", wr)
	}
}

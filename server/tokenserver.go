package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"tokenmanager/internal/config"
	_ "tokenmanager/internal/rpcjson"
	"tokenmanager/internal/tokenstore"
	pb "tokenmanager/token_management"
)

type readAck struct {
	resp *pb.ReadBroadcastResponse
	err  error
}

type TokenManagerServer struct {
	pb.UnimplementedTokenManagerServer

	cfg              *config.Config
	store            *tokenstore.Store
	self             config.ServerConfig
	requestTimeout   time.Duration
	sequence         atomic.Uint64
}

func (s *TokenManagerServer) WriteToken(ctx context.Context, in *pb.WriteTokenMsg) (*pb.WriteResponse, error) {
	if in == nil {
		return nil, errors.New("empty write request")
	}
	if !s.cfg.IsWriter(s.self.Name, in.TokenId) {
		return &pb.WriteResponse{Ack: 0, Message: "write authorization failed", TokenId: in.TokenId, Server: s.self.Name}, nil
	}

	wts := s.nextWTS()
	partialVal := in.PartialValue
	if partialVal == 0 {
		partialVal = in.Low + in.Mid + in.High
	}
	val := tokenstore.Value{
		WTS:        wts,
		PartialVal: partialVal,
		Name:       in.Name,
		Low:        in.Low,
		Mid:        in.Mid,
		High:       in.High,
	}

	s.store.UpsertIfNewer(in.TokenId, val)

	replicas := s.cfg.ReplicaServers(in.TokenId)
	if len(replicas) == 0 {
		return &pb.WriteResponse{Ack: 0, Message: "no replica nodes found for token", TokenId: in.TokenId, Server: s.self.Name}, nil
	}

	majority := len(replicas)/2 + 1
	ackCount := 0

	ackCh := make(chan int32, len(replicas))
	var wg sync.WaitGroup

	for _, peer := range replicas {
		if peer.Name == s.self.Name {
			ackCount++
			continue
		}

		wg.Add(1)
		go s.startBroadcast(&wg, peer, &pb.WriteBroadcastRequest{
			HashVal:     partialVal,
			ReadingFlag: false,
			Ack:         0,
			Wts:         wts,
			Name:        in.Name,
			Low:         in.Low,
			Mid:         in.Mid,
			High:        in.High,
			TokenId:     in.TokenId,
			Server:      s.self.Name,
		}, ackCh)
	}

	wg.Wait()
	close(ackCh)
	for ack := range ackCh {
		if ack > 0 {
			ackCount++
		}
	}

	msg := "write committed with quorum"
	ack := int32(1)
	if ackCount < majority {
		ack = 0
		msg = fmt.Sprintf("write failed quorum: got %d/%d acks", ackCount, majority)
	}

	return &pb.WriteResponse{
		Ack:      ack,
		Message:  msg,
		Wts:      wts,
		FinalVal: partialVal,
		Name:     in.Name,
		Low:      in.Low,
		Mid:      in.Mid,
		High:     in.High,
		TokenId:  in.TokenId,
		Server:   s.self.Name,
	}, nil
}

func (s *TokenManagerServer) ReadToken(ctx context.Context, in *pb.Token) (*pb.WriteResponse, error) {
	if in == nil {
		return nil, errors.New("empty read request")
	}
	if !s.cfg.IsReader(s.self.Name, in.TokenId) {
		return &pb.WriteResponse{Ack: 0, Message: "read authorization failed", TokenId: in.TokenId, Server: s.self.Name}, nil
	}

	if s.self.SleepMs > 0 {
		time.Sleep(time.Duration(s.self.SleepMs) * time.Millisecond)
	}

	replicas := s.cfg.ReplicaServers(in.TokenId)
	if len(replicas) == 0 {
		return &pb.WriteResponse{Ack: 0, Message: "no replica nodes found for token", TokenId: in.TokenId, Server: s.self.Name}, nil
	}

	best := tokenstore.Value{}
	if local, ok := s.store.Get(in.TokenId); ok {
		best = local
	}

	readCh := make(chan readAck, len(replicas))
	var wg sync.WaitGroup

	for _, peer := range replicas {
		if peer.Name == s.self.Name {
			continue
		}
		wg.Add(1)
		go s.startReadBroadcast(&wg, peer, in.TokenId, readCh)
	}

	wg.Wait()
	close(readCh)

	for item := range readCh {
		if item.err != nil || item.resp == nil {
			continue
		}
		candidate := tokenstore.Value{
			WTS:        item.resp.Wts,
			PartialVal: item.resp.FinalVal,
			Name:       item.resp.Name,
			Low:        item.resp.Low,
			Mid:        item.resp.Mid,
			High:       item.resp.High,
		}
		if candidate.WTS > best.WTS {
			best = candidate
		}
	}

	if best.WTS == "" {
		return &pb.WriteResponse{Ack: 0, Message: "token has no value yet", TokenId: in.TokenId, Server: s.self.Name}, nil
	}

	// Read-impose-write-all write-back.
	majority := len(replicas)/2 + 1
	ackCount := 0
	ackCh := make(chan int32, len(replicas))
	var wbWG sync.WaitGroup

	for _, peer := range replicas {
		if peer.Name == s.self.Name {
			s.store.UpsertIfNewer(in.TokenId, best)
			ackCount++
			continue
		}

		wbWG.Add(1)
		go s.startBroadcast(&wbWG, peer, &pb.WriteBroadcastRequest{
			HashVal:     best.PartialVal,
			ReadingFlag: true,
			Ack:         0,
			Wts:         best.WTS,
			Name:        best.Name,
			Low:         best.Low,
			Mid:         best.Mid,
			High:        best.High,
			TokenId:     in.TokenId,
			Server:      s.self.Name,
		}, ackCh)
	}

	wbWG.Wait()
	close(ackCh)
	for ack := range ackCh {
		if ack > 0 {
			ackCount++
		}
	}

	msg := "read served and write-back completed"
	ack := int32(1)
	if ackCount < majority {
		ack = 0
		msg = fmt.Sprintf("read served but write-back quorum failed: got %d/%d acks", ackCount, majority)
	}

	return &pb.WriteResponse{
		Ack:      ack,
		Message:  msg,
		Wts:      best.WTS,
		FinalVal: best.PartialVal,
		Name:     best.Name,
		Low:      best.Low,
		Mid:      best.Mid,
		High:     best.High,
		TokenId:  in.TokenId,
		Server:   s.self.Name,
	}, nil
}

func (s *TokenManagerServer) WriteBroadcast(ctx context.Context, in *pb.WriteBroadcastRequest) (*pb.WriteBroadcastResponse, error) {
	if in == nil {
		return nil, errors.New("empty write broadcast request")
	}
	if s.self.SleepMs > 0 {
		time.Sleep(time.Duration(s.self.SleepMs) * time.Millisecond)
	}
	if s.self.NegativeAck && !in.ReadingFlag {
		return &pb.WriteBroadcastResponse{Ack: 0}, nil
	}

	_, tokenKnown := s.cfg.TokenByID(in.TokenId)
	if !tokenKnown {
		return &pb.WriteBroadcastResponse{Ack: 0}, nil
	}

	applied := s.store.UpsertIfNewer(in.TokenId, tokenstore.Value{
		WTS:        in.Wts,
		PartialVal: in.HashVal,
		Name:       in.Name,
		Low:        in.Low,
		Mid:        in.Mid,
		High:       in.High,
	})
	if !applied {
		// No-op for stale timestamp still counts as successful delivery.
		return &pb.WriteBroadcastResponse{Ack: 1}, nil
	}
	return &pb.WriteBroadcastResponse{Ack: 1}, nil
}

func (s *TokenManagerServer) ReadBroadcast(ctx context.Context, in *pb.ReadBroadcastRequest) (*pb.ReadBroadcastResponse, error) {
	if in == nil {
		return nil, errors.New("empty read broadcast request")
	}
	if s.self.SleepMs > 0 {
		time.Sleep(time.Duration(s.self.SleepMs) * time.Millisecond)
	}
	v, ok := s.store.Get(in.TokenId)
	if !ok {
		return &pb.ReadBroadcastResponse{}, nil
	}
	return &pb.ReadBroadcastResponse{
		Wts:      v.WTS,
		FinalVal: v.PartialVal,
		Name:     v.Name,
		Low:      v.Low,
		Mid:      v.Mid,
		High:     v.High,
	}, nil
}

func (s *TokenManagerServer) startBroadcast(wg *sync.WaitGroup, peer config.ServerConfig, req *pb.WriteBroadcastRequest, ackCh chan<- int32) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	client, conn, err := s.clientForPeer(peer)
	if err != nil {
		ackCh <- 0
		return
	}
	defer conn.Close()

	resp, err := client.WriteBroadcast(ctx, req)
	if err != nil || resp == nil {
		ackCh <- 0
		return
	}
	ackCh <- resp.Ack
}

func (s *TokenManagerServer) startReadBroadcast(wg *sync.WaitGroup, peer config.ServerConfig, tokenID int32, readCh chan<- readAck) {
	defer wg.Done()

	ctx, cancel := context.WithTimeout(context.Background(), s.requestTimeout)
	defer cancel()

	client, conn, err := s.clientForPeer(peer)
	if err != nil {
		readCh <- readAck{err: err}
		return
	}
	defer conn.Close()

	resp, err := client.ReadBroadcast(ctx, &pb.ReadBroadcastRequest{TokenId: tokenID})
	readCh <- readAck{resp: resp, err: err}
}

func (s *TokenManagerServer) clientForPeer(peer config.ServerConfig) (pb.TokenManagerClient, *grpc.ClientConn, error) {
	addr := fmt.Sprintf("%s:%d", peer.Host, peer.Port)
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	if err != nil {
		return nil, nil, err
	}
	return pb.NewTokenManagerClient(conn), conn, nil
}

func (s *TokenManagerServer) nextWTS() string {
	n := time.Now().UnixNano()
	seq := s.sequence.Add(1)
	return fmt.Sprintf("%020d-%s-%06d", n, s.self.Name, seq)
}

func main() {
	var (
		port       = flag.Int("port", 50051, "server port")
		host       = flag.String("host", "127.0.0.1", "server host")
		configPath = flag.String("config", "token_config.yml", "path to YAML config")
		sleepMs    = flag.Int("sleep-ms", -1, "override fail-silent sleep latency in milliseconds")
		negAck     = flag.Bool("negative-ack", false, "force negative ack on write broadcasts")
	)
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed loading config: %v", err)
	}

	self, ok := cfg.ServerByPort(*port)
	if !ok {
		log.Fatalf("port %d not found in config servers", *port)
	}
	self.Host = *host
	if *sleepMs >= 0 {
		self.SleepMs = *sleepMs
	}
	if *negAck {
		self.NegativeAck = true
	}

	store := tokenstore.New()
	for _, t := range cfg.Tokens {
		for _, replica := range t.Replicas {
			if replica == self.Name {
				store.Set(t.ID, tokenstore.Value{})
				break
			}
		}
	}

	lis, err := net.Listen("tcp", fmt.Sprintf("%s:%d", *host, *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	grpcServer := grpc.NewServer()
	pb.RegisterTokenManagerServer(grpcServer, &TokenManagerServer{
		cfg:            cfg,
		store:          store,
		self:           self,
		requestTimeout: 3 * time.Second,
	})

	log.Printf("token server started: %s (%s:%d) sleep_ms=%d negative_ack=%v", self.Name, *host, *port, self.SleepMs, self.NegativeAck)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("grpc serve failed: %v", err)
	}
}

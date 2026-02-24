package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	_ "tokenmanager/internal/rpcjson"
	pb "tokenmanager/token_management"
)

func main() {
	var (
		write       = flag.Bool("write", false, "send write request")
		read        = flag.Bool("read", false, "send read request")
		host        = flag.String("host", "127.0.0.1", "server host")
		port        = flag.Int("port", 50051, "server port")
		tokenID     = flag.Int("id", 1, "token id")
		name        = flag.String("name", "", "name value for writes")
		partialVal  = flag.Uint64("partial", 0, "partial value for writes (optional)")
		low         = flag.Uint64("low", 0, "low value for writes")
		mid         = flag.Uint64("mid", 0, "mid value for writes")
		high        = flag.Uint64("high", 0, "high value for writes")
		timeoutSec  = flag.Int("timeout", 5, "request timeout in seconds")
	)
	flag.Parse()

	if !*write && !*read {
		log.Fatal("choose one of -write or -read")
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	conn, err := grpc.Dial(
		addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.CallContentSubtype("json")),
	)
	if err != nil {
		log.Fatalf("dial failed: %v", err)
	}
	defer conn.Close()

	client := pb.NewTokenManagerClient(conn)
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(*timeoutSec)*time.Second)
	defer cancel()

	if *write {
		resp, err := client.WriteToken(ctx, &pb.WriteTokenMsg{
			TokenId:      int32(*tokenID),
			PartialValue: *partialVal,
			Name:         *name,
			Low:          *low,
			Mid:          *mid,
			High:         *high,
		})
		if err != nil {
			log.Fatalf("write failed: %v", err)
		}
		printResp("WRITE", resp)
		return
	}

	resp, err := client.ReadToken(ctx, &pb.Token{TokenId: int32(*tokenID)})
	if err != nil {
		log.Fatalf("read failed: %v", err)
	}
	printResp("READ", resp)
}

func printResp(kind string, resp *pb.WriteResponse) {
	if resp == nil {
		fmt.Printf("%s response is nil\n", kind)
		return
	}
	fmt.Printf("%s ACK=%d SERVER=%s TOKEN=%d\n", kind, resp.Ack, resp.Server, resp.TokenId)
	fmt.Printf("message: %s\n", resp.Message)
	fmt.Printf("wts=%s partial=%d name=%q low=%d mid=%d high=%d\n", resp.Wts, resp.FinalVal, resp.Name, resp.Low, resp.Mid, resp.High)
}

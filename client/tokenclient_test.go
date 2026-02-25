package main

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	pb "tokenmanager/token_management"
)

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()

	old := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("create pipe: %v", err)
	}
	os.Stdout = w

	defer func() {
		os.Stdout = old
	}()

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("read captured stdout: %v", err)
	}
	if err := r.Close(); err != nil {
		t.Fatalf("close reader: %v", err)
	}

	return buf.String()
}

func TestPrintRespNil(t *testing.T) {
	out := captureStdout(t, func() {
		printResp("READ", nil)
	})
	if !strings.Contains(out, "READ response is nil") {
		t.Fatalf("unexpected output: %q", out)
	}
}

func TestPrintRespFormatsFields(t *testing.T) {
	resp := &pb.WriteResponse{
		Ack:      1,
		Server:   "s1",
		TokenId:  7,
		Message:  "ok",
		Wts:      "w-1",
		FinalVal: 42,
		Name:     "abc",
		Low:      1,
		Mid:      2,
		High:     3,
	}
	out := captureStdout(t, func() {
		printResp("WRITE", resp)
	})

	required := []string{
		"WRITE ACK=1 SERVER=s1 TOKEN=7",
		"message: ok",
		`wts=w-1 partial=42 name="abc" low=1 mid=2 high=3`,
	}
	for _, part := range required {
		if !strings.Contains(out, part) {
			t.Fatalf("missing %q in output: %q", part, out)
		}
	}
}

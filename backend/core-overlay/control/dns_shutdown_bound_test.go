package control

import (
	"context"
	"errors"
	"net"
	"syscall"
	"testing"
)

func TestDNSDrainTimeoutReleasesRealBind(t *testing.T) {
	conn, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil { t.Fatal(err) }
	address := conn.LocalAddr().String()
	err = stopDNSBind(context.Background(), func(context.Context) error { return context.DeadlineExceeded }, conn.Close)
	if err != nil { t.Fatal(err) }
	replacement, err := net.ListenPacket("udp4", address)
	if err != nil { t.Fatalf("old bind was not released: %v", err) }
	replacement.Close()
}

func TestDNSDrainPreservesCloseAndOtherErrors(t *testing.T) {
	for _, item := range []struct{ drain, close, want error }{
		{context.DeadlineExceeded, syscall.EIO, syscall.EIO},
		{syscall.EINVAL, nil, syscall.EINVAL},
		{nil, net.ErrClosed, nil},
	} {
		err := stopDNSBind(context.Background(), func(context.Context) error { return item.drain }, func() error { return item.close })
		if !errors.Is(err, item.want) { t.Fatalf("error changed: got=%v want=%v", err, item.want) }
	}
}

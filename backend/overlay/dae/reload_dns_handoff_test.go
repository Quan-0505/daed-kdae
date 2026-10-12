package dae

import (
	"io"
	"net"
	"testing"

	"github.com/daeuniverse/dae/control"
	"github.com/sirupsen/logrus"
)

// Exercise real DNS sockets: cloning the transparent-proxy listener cannot
// release this separate bind. A rejected replacement must permit restoration.
func TestNativeDNSBindHandoffAndRollback(t *testing.T) {
	probe, err := net.Listen("tcp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	address := probe.Addr().String()
	probe.Close()
	log := logrus.New()
	log.SetOutput(io.Discard)
	previous, err := control.NewDNSListener(log, "tcp+udp://"+address, nil)
	if err != nil {
		t.Fatal(err)
	}
	candidate, err := control.NewDNSListener(log, "tcp+udp://"+address, nil)
	if err != nil {
		t.Fatal(err)
	}
	defer previous.Stop()
	defer candidate.Stop()
	if err := previous.Start(); err != nil {
		t.Fatal(err)
	}
	if err := candidate.Start(); err == nil {
		t.Fatal("candidate unexpectedly acquired an occupied DNS bind")
	}
	if err := previous.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := candidate.Start(); err != nil {
		t.Fatalf("candidate could not acquire the released bind: %v", err)
	}
	// Reject the new generation after it has acquired its socket.
	if err := candidate.Stop(); err != nil {
		t.Fatal(err)
	}
	if err := previous.Start(); err != nil {
		t.Fatalf("previous DNS server could not be restored: %v", err)
	}
}

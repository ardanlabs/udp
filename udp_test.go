package udp_test

import (
	"bytes"
	"io"
	"net"
	"os"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ardanlabs/udp"
)

// TestUDP provide a test of listening for a connection and
// echoing the data back.
func TestUDP(t *testing.T) {
	resetLog()
	defer displayLog()

	t.Log("Given the need to listen and process UDP data.")
	{
		// Create a configuration.
		cfg := udp.Config{
			NetType: "udp4",
			Addr:    ":0",

			ConnHandler: udpConnHandler{},
			ReqHandler:  udpReqHandler{},
			RespHandler: udpRespHandler{},
		}

		// Create a new UDP value.
		u, err := udp.New("TEST", cfg)
		if err != nil {
			t.Fatal("\tShould be able to create a new UDP listener.", failed, err)
		}
		t.Log("\tShould be able to create a new UDP listener.", success)

		// Start accepting client data.
		if err := u.Start(); err != nil {
			t.Fatal("\tShould be able to start the UDP listener.", failed, err)
		}
		t.Log("\tShould be able to start the UDP listener.", success)

		defer u.Stop()

		// Let's connect back and send a UDP package
		conn, err := net.Dial("udp4", u.Addr().String())
		if err != nil {
			t.Fatal("\tShould be able to dial a new UDP connection.", failed, err)
		}
		t.Log("\tShould be able to dial a new UDP connection.", success)

		// Send some know data to the udp listener.
		b := bytes.NewBuffer([]byte{0x01, 0x3D, 0x06, 0x00, 0x58, 0x68, 0x9b, 0x9d, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0xFF, 0xFC, 0x00, 0x01})
		b.WriteTo(conn)

		// Setup a limit reader to extract the response.
		lr := io.LimitReader(conn, 6)

		// Let's read the response.
		data := make([]byte, 6)
		if _, err := lr.Read(data); err != nil {
			t.Fatal("\tShould be able to read the response from the connection.", failed, err)
		}
		t.Log("\tShould be able to read the response from the connection.", success)

		response := string(data)

		if response == "GOT IT" {
			t.Log("\tShould receive the string \"GOT IT\".", success)
		} else {
			t.Error("\tShould receive the string \"GOT IT\".", failed, response)
		}

		d := atomic.LoadInt64(&dur)
		duration := time.Duration(d)

		if duration <= 2*time.Second {
			t.Log("\tShould be less that 2 seconds.", success)
		} else {
			t.Error("\tShould be less that 2 seconds.", failed, duration)
		}
	}
}

// Test udp.Addr works correctly.
func TestUDPAddr(t *testing.T) {
	resetLog()
	defer displayLog()

	t.Log("Given the need to listen on any port and know that bound UDP address.")
	{
		// Create a configuration.
		cfg := udp.Config{
			NetType: "udp4",
			Addr:    ":0", // Defer port assignment to OS.

			ConnHandler: udpConnHandler{},
			ReqHandler:  udpReqHandler{},
			RespHandler: udpRespHandler{},
		}

		// Create a new UDP value.
		u, err := udp.New("TEST", cfg)
		if err != nil {
			t.Fatal("\tShould be able to create a new UDP listener.", failed, err)
		}
		t.Log("\tShould be able to create a new UDP listener.", success)

		// Addr should be nil before start.
		if addr := u.Addr(); addr != nil {
			t.Fatalf("\tAddr() should be nil before Start; Addr() = %q. %s", addr, failed)
		}
		t.Log("\tAddr() should be nil before Start.", success)

		// Start accepting client data.
		if err := u.Start(); err != nil {
			t.Fatal("\tShould be able to start the UDP listener.", failed, err)
		}
		defer u.Stop()

		// Addr should be non-nil after Start.
		addr := u.Addr()
		if addr == nil {
			t.Fatal("\tAddr() should be not be nil after Start.", failed)
		}
		t.Log("\tAddr() should be not be nil after Start.", success)

		// The OS should assign a random open port, which shouldn't be 0.
		_, port, err := net.SplitHostPort(addr.String())
		if err != nil {
			t.Fatalf("\tSplitHostPort should not fail. failed %v. %s", err, failed)
		}
		if port == "0" {
			t.Fatalf("\tAddr port should not be %q. %s", port, failed)
		}
		t.Logf("\tAddr() should be not be 0 after Start (port = %q). %s", port, success)
	}
}

// Test generic UDP write timeout.
func TestUDPWriteTimeout(t *testing.T) {
	t.Log("Given the need to get a timeout error on UDP write.")
	{
		localAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:0")
		if err != nil {
			t.Fatalf("Should not have failed to resolve UDP address. Err[%v] %s", err, failed)
		}
		remoteAddr, err := net.ResolveUDPAddr("udp4", "127.0.0.1:1234")
		if err != nil {
			t.Fatalf("Should not have failed to resolve UDP address. Err[%v] %s", err, failed)
		}

		conn, err := net.ListenUDP("udp4", localAddr)
		if err != nil {
			t.Fatalf("Should not have failed to create UDP connection. Err[%v] %s", err, failed)
		}

		if err := conn.SetWriteDeadline(time.Now().Add(1)); err != nil {
			t.Fatalf("Should not have failed to set write deadline. Err[%v] %s", err, failed)
		}

		const str = "String to send via UDP socket for testing purposes."

		_, err = conn.WriteToUDP([]byte(str), remoteAddr)
		if err == nil {
			t.Fatalf("Should not gotten an error %s", failed)
		}

		t.Logf("Got error [%T: %v] %s", err, err, success)

		opError, ok := err.(*net.OpError)
		if !ok {
			t.Fatalf("Should have gotten *net.OpError, got [%T: %v] %s", err, err, failed)
		}

		if !opError.Timeout() {
			t.Fatalf("Should have gotten a timeout error %s", failed)
		}

		t.Logf("Got timeout error %s", success)
	}
}

// =============================================================================

// Success and failure markers.
var (
	success = "\u2713"
	failed  = "\u2717"
)

// logdash is the central buffer where all logs are stored.
var logdash bytes.Buffer

// resetLog resets the contents of Logdash.
func resetLog() {
	logdash.Reset()
}

// displayLog writes the Logdash data to standand out, if testing in verbose mode
// was turned on.
func displayLog() {
	if !testing.Verbose() {
		return
	}

	logdash.WriteTo(os.Stdout)
}

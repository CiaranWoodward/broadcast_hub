package server

import (
	"log"
	"net"
	"testing"
	"time"

	"github.com/CiaranWoodward/broadcast_hub/client"
	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

const (
	byte_time_1kbps = time.Millisecond
)

// Make a connection slow, to simulate real-ish connection behaviour
func makeSlow(con net.Conn, byte_time time.Duration) net.Conn {
	in, out := net.Pipe()

	forwarder := func(in, out net.Conn) {
		for {
			buffer := make([]byte, 10)
			n, err := in.Read(buffer)
			if err != nil {
				out.Close()
				break
			}
			buffer = buffer[:n]
			<-time.After(time.Millisecond * time.Duration(n))
			n2, err := out.Write(buffer)
			if n2 != n || err != nil {
				in.Close()
				break
			}
		}
	}

	go forwarder(con, in)
	go forwarder(in, con)

	return out
}

func TestSlowClient(t *testing.T) {
	// Test that a slow client won't be overloaded by fast neighbors
	defer goleak.VerifyNone(t)

	// Real Server
	server := NewServer()

	// Create the fast client
	cli, ser := net.Pipe()
	client_fast := client.NewClient(cli)
	server.AddClientByConnection(ser)
	fast_cid, status := client_fast.GetClientId()
	assert.Equal(t, msg.SUCCESS, status)

	// Create the slow client
	cli, ser = net.Pipe()
	cli = makeSlow(cli, byte_time_1kbps)
	client_slow := client.NewClient(cli)
	server.AddClientByConnection(ser)
	slow_cid, status := client_slow.GetClientId()
	assert.Equal(t, msg.SUCCESS, status)

	// Make a long message to send
	longMessage := make([]byte, 1000)
	for i := 0; i < 1000; i++ {
		longMessage[i] = byte(i % 256)
	}

	// Forever receive all relays into the bitbucket
	receiver := func(cli *client.Client) {
		for {
			_, ok := <-cli.Relays
			if !ok {
				break
			}
		}
	}
	go receiver(client_fast)
	go receiver(client_slow)

	// Send a message in each direction to check that works
	csm, status := client_fast.RelayMessage(longMessage, []msg.ClientId{slow_cid})
	assert.Len(t, csm, 0)
	assert.Equal(t, msg.SUCCESS, status)
	csm, status = client_slow.RelayMessage(longMessage, []msg.ClientId{fast_cid})
	assert.Len(t, csm, 0)
	assert.Equal(t, msg.SUCCESS, status)

	// Send 15 messages from fast to slow with 1ms spacing, record statuses
	n_messages := 15
	statuses := make([]msg.Status, n_messages)
	for i := 0; i < n_messages; i++ {
		csm, status := client_fast.RelayMessage(longMessage, []msg.ClientId{slow_cid})
		assert.Equal(t, msg.SUCCESS, status)
		status, ok := csm[slow_cid]
		if ok {
			statuses[i] = status
		} else {
			statuses[i] = msg.SUCCESS
		}
		<-time.After(time.Millisecond * 400)
	}
	log.Print(statuses)

	// Look through the statuses. We want to see a transition from SUCCESS -> NO_BUFFER
	// And also a successful recovery from NO_BUFFER -> SUCCESS
	var throttled, recovered bool
	prev := statuses[0]
	for i := 1; i < len(statuses); i++ {
		if prev == msg.SUCCESS && statuses[i] == msg.NO_BUFFER {
			throttled = true
		}
		if prev == msg.NO_BUFFER && statuses[i] == msg.SUCCESS {
			recovered = true
		}
		if statuses[i] != msg.SUCCESS && statuses[i] != msg.NO_BUFFER {
			t.Errorf("Unexpected status: %d", statuses[i])
		}
		prev = statuses[i]
	}
	assert.True(t, throttled)
	assert.True(t, recovered)
	server.Close()
}

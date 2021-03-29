package server

import (
	"net"
	"sync"
	"testing"

	"github.com/CiaranWoodward/broadcast_hub/client"
	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/stretchr/testify/assert"
)

func TestServerAndClient(t *testing.T) {
	// Run through a basic example using both server and client

	// Real Server
	server := NewServer()
	cid_chan := make(chan msg.ClientId)

	// Create 100 clients and connect them in parallel
	n_clients := 100
	wg := sync.WaitGroup{}
	for i := 0; i < n_clients; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			cli, ser := net.Pipe()
			server.AddClientByConnection(ser)
			tc := client.NewClient(cli)
			cid, status := tc.GetClientId()
			assert.Equal(t, msg.SUCCESS, status)
			// Output the cid that was obtained for uniqueness checking
			cid_chan <- cid

			// Verify we can get relay indications
			assert.Equal(t, []byte{1, 2, 3, 4, 5}, (<-tc.Relays).Msg)
		}()
	}

	// Verify each CID is unique
	cid_set := make(map[msg.ClientId]struct{})
	for i := 0; i < n_clients; i++ {
		new_cid := <-cid_chan
		_, exists := cid_set[new_cid]
		assert.False(t, exists, "Duplicate ID: %d", new_cid)
		cid_set[new_cid] = struct{}{}
	}

	// Add a final client that we will use for the list req
	cli, ser := net.Pipe()
	server.AddClientByConnection(ser)
	tc := client.NewClient(cli)
	cids, status := tc.ListOtherClients()
	assert.Equal(t, msg.SUCCESS, status)

	for _, cid := range cids {
		_, exists := cid_set[cid]
		assert.True(t, exists, "ID %d not in returned list", cid)
	}

	//Send a relay message to all other clients, plus one invalid one
	//Verify that it is correctly relayed to the non-invalid IDs
	invalid_id := msg.ClientId(0x7621a3c5418eb972)
	cids = append(cids, invalid_id)
	csm, status := tc.RelayMessage([]byte{1, 2, 3, 4, 5}, cids)
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, len(csm), 1)
	assert.Equal(t, msg.INVALID_ID, csm[invalid_id])

	wg.Wait()
}

func TestServerListener(t *testing.T) {
	// Test the listener functionality using a TCP connection
	server := NewServer()
	listener, err := net.ListenTCP("tcp", &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1)}) // Using ephemeral port (0) for server is only suitable for testing.
	assert.Nil(t, err)
	serverAddr := listener.Addr().String()
	server.AddListener(listener)

	n_client := 10
	wg_setup := sync.WaitGroup{}
	wg_done := sync.WaitGroup{}
	wg_setup.Add(n_client)
	wg_done.Add(n_client)
	for i := 0; i < n_client; i++ {
		go func() {
			defer wg_done.Done()

			conn, err := net.Dial("tcp", serverAddr)
			assert.Nil(t, err)
			tc := client.NewClient(conn)

			// Verify connection
			_, status := tc.GetClientId()
			assert.Equal(t, status, msg.SUCCESS)

			wg_setup.Done()

			// Verify we can get relay indications
			assert.Equal(t, []byte{255, 0}, (<-tc.Relays).Msg)

			tc.Close()
		}()
	}

	// Wait for all of the clients to connect
	wg_setup.Wait()

	// Start another client and send a relay message to all of the others
	conn, err := net.Dial("tcp", serverAddr)
	assert.Nil(t, err)
	tc := client.NewClient(conn)
	cids, status := tc.ListOtherClients()
	assert.Equal(t, msg.SUCCESS, status)
	csm, status := tc.RelayMessage([]byte{255, 0}, cids)
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, 0, len(csm))

	// Wait for all of the clients to exit
	wg_done.Wait()
	server.Close()
}

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

	// Real Server running in seperate goroutine
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
			server.addClientByConnection(ser)
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
	server.addClientByConnection(ser)
	tc := client.NewClient(cli)
	cids, status := tc.ListOtherClients()
	assert.Equal(t, msg.SUCCESS, status)

	for _, cid := range cids {
		_, exists := cid_set[cid]
		assert.True(t, exists, "ID %d not in returned list", cid)
	}

	//Send a relay message to all other clients
	csm, status := tc.RelayMessage([]byte{1, 2, 3, 4, 5}, cids)
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, len(csm), 0)

	wg.Wait()
}

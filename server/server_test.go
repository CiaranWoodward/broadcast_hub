package server

import (
	"net"
	"testing"

	"github.com/CiaranWoodward/broadcast_hub/client"
	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/stretchr/testify/assert"
)

func TestIdReq(t *testing.T) {

	// Real Server running in seperate goroutine
	server := NewServer()
	cid_chan := make(chan msg.ClientId)

	// Create 100 clients and connect them in parallel
	n_clients := 100
	for i := 0; i < n_clients; i++ {
		go func() {
			cli, ser := net.Pipe()
			server.addClientByConnection(ser)
			tc := client.NewClient(cli)
			cid, status := tc.GetClientId()
			assert.Equal(t, msg.SUCCESS, status)
			// Output the cid that was obtained for uniqueness checking
			cid_chan <- cid
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
}

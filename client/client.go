/*
Package client implements the user-facing API of a broadcast_hub client.
*/
package client

import (
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CiaranWoodward/broadcast_hub/msg"
)

// Client struct - instatiated with the 'NewClient' Function.
type client struct {
	// Message transcoders
	tc msg.Transcoder
	dc msg.StreamDecoder
	// Internal message ID counter (for unique IDs)
	mid uint32
	// Internal connection state
	con net.Conn
	// Map of message IDs to the channel waiting for the response, and a mutex protecting it
	mid_map       map[uint32]chan msg.Message
	mid_map_mutex sync.Mutex
}

// NewClient creates a new client, for use with the methods in this package
// Returns pointer to the instantiated client struct
// When work with the client struct is complete, the 'Close' Method must be called
// Passes ownership of the Conn to the client, which will handle closing of it (Is this a good idea?)
func NewClient(con net.Conn) *client {
	c := client{
		tc:      &msg.CborTranscoder{},
		dc:      msg.NewCborStreamDecoder(con),
		mid:     0,
		con:     con,
		mid_map: make(map[uint32]chan msg.Message),
	}
	c.startDispatcher()
	return &c
}

// Get a new unique message ID. Can be safely accessed by different goroutines.
func (c *client) getMessageId() uint32 {
	return atomic.AddUint32(&c.mid, 1)
}

func (c *client) addResponseChannel(mid uint32) chan msg.Message {
	ch := make(chan msg.Message)
	c.mid_map_mutex.Lock()
	c.mid_map[mid] = ch
	c.mid_map_mutex.Unlock()
	return ch
}

func (c *client) removeResponseChannel(mid uint32) {
	c.mid_map_mutex.Lock()
	delete(c.mid_map, mid)
	c.mid_map_mutex.Unlock()
}

func (c *client) sendToResponseChannel(m msg.Message) {
	c.mid_map_mutex.Lock()
	ch := c.mid_map[m.MessageId]
	c.mid_map_mutex.Unlock()
	if ch != nil {
		ch <- m
	}
}

func (c *client) closeAllResponseChannels() {
	c.mid_map_mutex.Lock()
	for _, ch := range c.mid_map {
		close(ch)
	}
	c.mid_map_mutex.Unlock()
}

func (c *client) sendMessage(m msg.Message) msg.Status {
	encoded_req, ok := c.tc.Encode(m)
	if !ok {
		return msg.ENCODING_ERROR
	}
	n, err := c.con.Write(encoded_req)
	if (err != nil) || (n != len(encoded_req)) {
		return msg.CONNECTION_ERROR
	}
	return msg.SUCCESS
}

func (c *client) startDispatcher() {
	go func() {
		// Reader function to pull from
		for {
			//TODO: Test that cleanup works as expected
			msgout, ok := c.dc.DecodeNext()
			if ok {
				c.sendToResponseChannel(msgout)
			} else {
				c.closeAllResponseChannels()
				break
			}
		}
	}()
}

// Identity Message
// GetClientId gets the ID of the client from the server.
// Returns a channel that will have this client's ID sent into it
func (c *client) GetClientId() (clientid msg.ClientId, status msg.Status) {
	// Form the message
	mid := c.getMessageId()
	req := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mid,
		IdReq:     &msg.IdentifyRequest{},
	}

	// Create a channel for receiving the response. Defer cleaning it up.
	rsp_chan := c.addResponseChannel(mid)
	defer c.removeResponseChannel(mid)

	//Encode the request and send it over the connection
	c.sendMessage(req)

	// Wait for response, or time out
	for {
		select {
		case rsp, ok := <-rsp_chan:
			if !ok {
				return 0, msg.CONNECTION_ERROR
			}
			if rsp.IdRes == nil {
				return 0, msg.ENCODING_ERROR
			}
			return rsp.IdRes.Id, msg.SUCCESS

		case <-time.After(5 * time.Second):
			return 0, msg.TIMEOUT
		}
	}
}

// List Message
// ListOtherClients gets a list of all other nodes connected to the server.
// Returns a channel that will have the other client IDs individually streamed into it
func (*client) ListOtherClients() (clientid []msg.ClientId, status msg.Status) {
	//TODO: Stub
	return []msg.ClientId{}, msg.SUCCESS
}

// Relay Message
// RelayMessage sends a message to be relayed to other clients by the server.
// Maximum length of the message is 1024 bytes
// Maximum length of clients is 255
func (*client) RelayMessage(message []byte, clients []msg.ClientId) (relayStatus msg.ClientStatusMap, status msg.Status) {
	//TODO: Stub
	return
}

// Close closes a client, and its associated resources
func (c *client) Close() {
	c.con.Close()
}

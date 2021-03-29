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

// Length of the buffered channel for holding incoming relays
const internalMessageBufferSize = 10

// Client struct - instatiated with the 'NewClient' Function.
type Client struct {
	// Channel to receive incoming relay indications
	Relays chan msg.RelayIndication
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

// NewClient creates a new client, for use with the methods in this package.
// Returns pointer to the instantiated client.
//
// The application should be sure to continually process items in the 'Relays' channel,
// so as not to fill the internal buffer.
//
// When work with the client is complete, the 'Close' Method should be called, which will
// handle releasing of all resources, including the 'con' argument.
func NewClient(con net.Conn) *Client {
	tc := &msg.CborTranscoder{}
	c := Client{
		Relays:  make(chan msg.RelayIndication, internalMessageBufferSize),
		tc:      tc,
		dc:      tc.NewStreamDecoder(con),
		mid:     0,
		con:     con,
		mid_map: make(map[uint32]chan msg.Message),
	}
	c.startDispatcher()
	return &c
}

// GetClientId gets the ID of the client from the server. This is the 'Identity Message'.
// Returns a channel that will have this client's ID sent into it
func (c *Client) GetClientId() (clientid msg.ClientId, status msg.Status) {
	// Form the message
	req := c.newMessage()
	req.IdReq = &msg.IdentifyRequest{}

	// Create a channel for receiving the response. Defer cleaning it up.
	rsp_chan := c.addResponseChannel(req.MessageId)
	defer c.removeResponseChannel(req.MessageId)

	//Encode the request and send it over the connection
	status = c.sendMessage(req)
	if status != msg.SUCCESS {
		return 0, status
	}

	// Wait for response, or time out
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

// ListOtherClients gets a list of all other nodes connected to the server. This is the 'List Message'.
// Returns a channel that will have the other client IDs individually streamed into it
func (c *Client) ListOtherClients() (clientid []msg.ClientId, status msg.Status) {
	// Form the message
	req := c.newMessage()
	req.ListReq = &msg.ListRequest{}

	// Create a channel for receiving the response. Defer cleaning it up.
	rsp_chan := c.addResponseChannel(req.MessageId)
	defer c.removeResponseChannel(req.MessageId)

	//Encode the request and send it over the connection
	status = c.sendMessage(req)
	if status != msg.SUCCESS {
		return
	}

	// Wait for response, or time out
	select {
	case rsp, ok := <-rsp_chan:
		if !ok {
			status = msg.CONNECTION_ERROR
			return
		}
		if rsp.ListRes == nil {
			status = msg.ENCODING_ERROR
			return
		}
		return rsp.ListRes.Others, msg.SUCCESS

	case <-time.After(5 * time.Second):
		status = msg.TIMEOUT
		return
	}
}

// RelayMessage sends a message to be relayed to other clients by the server. This is the 'Relay Message'.
//
// Maximum length of the message is 1024 bytes.
// Maximum length of clients is 255.
//
// The returned clientStatusMap is only valid if status == SUCCESS
// The returned clientStatusMap does not include the client IDs of successfully relayed messages - they are omitted for efficiency
func (c *Client) RelayMessage(message []byte, clients []msg.ClientId) (relayStatus msg.ClientStatusMap, status msg.Status) {
	// Check protocol parameters
	if len(message) > 1024 || len(clients) > 255 {
		status = msg.TOO_LONG
		return
	}
	// Form the message
	req := c.newMessage()
	req.RelayReq = &msg.RelayRequest{Dest: clients, Msg: message}

	// Create a channel for receiving the response. Defer cleaning it up.
	rsp_chan := c.addResponseChannel(req.MessageId)
	defer c.removeResponseChannel(req.MessageId)

	//Encode the request and send it over the connection
	status = c.sendMessage(req)
	if status != msg.SUCCESS {
		return
	}

	// Wait for response, or time out
	select {
	case rsp, ok := <-rsp_chan:
		if !ok {
			status = msg.CONNECTION_ERROR
			return
		}
		if rsp.RelayRes == nil {
			status = msg.ENCODING_ERROR
			return
		}
		return rsp.RelayRes.StatusMap, rsp.RelayRes.Status

	case <-time.After(5 * time.Second):
		status = msg.TIMEOUT
		return
	}
}

// Close closes a client, and its associated resources
func (c *Client) Close() {
	c.con.Close()
}

// Get a new base message with unique message ID. Can be safely accessed by different goroutines.
func (c *Client) newMessage() msg.Message {
	return msg.Message{
		Version:   msg.MyVersion,
		MessageId: atomic.AddUint32(&c.mid, 1),
	}
}

func (c *Client) addResponseChannel(mid uint32) chan msg.Message {
	ch := make(chan msg.Message)
	c.mid_map_mutex.Lock()
	c.mid_map[mid] = ch
	c.mid_map_mutex.Unlock()
	return ch
}

func (c *Client) removeResponseChannel(mid uint32) {
	c.mid_map_mutex.Lock()
	delete(c.mid_map, mid)
	c.mid_map_mutex.Unlock()
}

// Only to be called by dispatcher
func (c *Client) sendToResponseChannel(m msg.Message) {
	c.mid_map_mutex.Lock()
	ch, ok := c.mid_map[m.MessageId]
	c.mid_map_mutex.Unlock()
	if ok {
		ch <- m
	}
}

// Only to be called by dispatcher
func (c *Client) closeAllResponseChannels() {
	c.mid_map_mutex.Lock()
	for _, ch := range c.mid_map {
		close(ch)
	}
	c.mid_map_mutex.Unlock()
}

// Encode and transmit a message to the server
func (c *Client) sendMessage(m msg.Message) msg.Status {
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

func (c *Client) startDispatcher() {
	go func() {
		// Read messages from the transport, and dispatch them to the relevant requester
		for {
			msgout, ok := c.dc.DecodeNext()
			if ok {
				if msgout.RelayInd != nil {
					// Relay indication (This WILL block if the application isn't servicing the channel)
					c.Relays <- *msgout.RelayInd
				} else {
					// Response message
					c.sendToResponseChannel(msgout)
				}
			} else {
				c.closeAllResponseChannels()
				break
			}
		}
		close(c.Relays)
	}()
}

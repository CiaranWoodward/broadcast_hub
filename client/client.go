/*
Package client implements the user-facing API of a broadcast_hub client.
*/
package client

import (
	"bufio"
	"net"

	"github.com/CiaranWoodward/broadcast_hub/msg"
)

// Client struct - instatiated with the 'NewClient' Function.
type client struct {
	tc  msg.Transcoder
	dc  msg.StreamDecoder
	con *net.Conn
}

// NewClient creates a new client, for use with the methods in this package
// Returns pointer to the instantiated client struct
// When work with the client struct is complete, the 'Close' Method must be called
// Passes ownership of the Conn to the client, which will handle closing of it (Is this a good idea?)
func NewClient(con *net.Conn) *client {
	return &client{
		tc:  &msg.CborTranscoder{},
		dc:  msg.NewCborStreamDecoder(bufio.NewReader(*con)),
		con: con,
	}
}

// Identity Message
// GetClientId gets the ID of the client from the server.
// Returns a channel that will have this client's ID sent into it
func (*client) GetClientId() (clientid msg.ClientId, status msg.Status) {
	return 1, msg.SUCCESS
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
}

// Close closes a client, and its associated resources
func (c *client) Close() {
	(*c.con).Close()
}

/*
Package server implements the user-facing API of a broadcast_hub server.
*/
package server

import (
	"net"
	"sync"

	"github.com/CiaranWoodward/broadcast_hub/msg"
)

// server representation of a connected client
type serverClient struct {
	// Message stream decoder
	dc msg.StreamDecoder
	// Internal connection state
	con net.Conn
	// Client Id
	cid msg.ClientId
}

type Server struct {
	// Internal client ID counter (for unique IDs)
	cid uint32
	// Map of all connected clients
	clients       map[msg.ClientId]serverClient
	clients_mutex sync.Mutex
}

func NewServer() *Server {
	return &Server{}
}

// Add a listener which will accept new incoming connections automatically
func (s *Server) AddListener(l net.Listener) {

}

// Close the server, and all associated resources and connections
func (s *Server) Close() {

}

// Start the dispatcher that will handle each received message
func (s *Server) startDispatcher(sc *serverClient) {

}

// Handle an incoming ID Request Message
func (s *Server) handleIdRequest(sc *serverClient, mesg *msg.Message) {

}

// Handle an incoming List Request Message
func (s *Server) handleListRequest(sc *serverClient, mesg *msg.Message) {

}

// Handle an incoming Relay Request Message
// This one is trickier.
func (s *Server) handleRelayRequest(sc *serverClient, *mesg msg.Message) {

}

// Add a new client connection
// TODO: Do we need to export this for tests to be able to use it?
func (s *Server) addClientByConnection(c net.Conn) {
	// Generate CID, add it to the map
}

// Remove a client from server mapping
func (s *Server) removeClient(cid msg.ClientId) {

}

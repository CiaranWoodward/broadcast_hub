/*
Package server implements the user-facing API of a broadcast_hub server.
*/
package server

import (
	"net"
	"sync"
	"sync/atomic"

	"github.com/CiaranWoodward/broadcast_hub/msg"
)

// server representation of a connected client
type serverClient struct {
	// Client Id
	cid msg.ClientId
	// Message stream decoder
	tc msg.Transcoder
	dc msg.StreamDecoder
	// Internal connection state
	con net.Conn
}

type Server struct {
	// Internal client ID counter (for unique IDs)
	cid msg.ClientId
	// Map of all connected clients
	clients       map[msg.ClientId]serverClient
	clients_mutex sync.RWMutex
}

func NewServer() *Server {
	return &Server{
		clients: make(map[msg.ClientId]serverClient),
	}
}

// Add a listener which will accept new incoming connections automatically
func (s *Server) AddListener(l net.Listener) {

}

// Close the server, and all associated resources and connections
func (s *Server) Close() {

}

// Start the dispatcher that will handle each received message
func (s *Server) startDispatcher(sc serverClient) {
	go func() {
		// Read messages from the transport, and dispatch them to the relevant handler
		// Currently the server will only handle a single request per connected client (A fair restriction for a low-bandwidth protocol like this)
		for {
			msgout, ok := sc.dc.DecodeNext()
			if ok {
				if msgout.IdReq != nil {
					s.handleIdRequest(&sc, &msgout)
				}
				if msgout.ListReq != nil {
					s.handleListRequest(&sc, &msgout)
				}
				if msgout.RelayReq != nil {
					s.handleRelayRequest(&sc, &msgout)
				}
			} else {
				s.removeClient(sc.cid)
				break
			}
		}
	}()
}

// Handle an incoming ID Request Message
func (s *Server) handleIdRequest(sc *serverClient, mesg *msg.Message) msg.Status {
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		IdRes: &msg.IdentifyResponse{
			Id: sc.cid,
		},
	}
	return sc.sendMessage(rsp)
}

// Handle an incoming List Request Message (This has a potentially unbounded response size. Limited by number of connected clients.)
func (s *Server) handleListRequest(sc *serverClient, mesg *msg.Message) msg.Status {
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		ListRes: &msg.ListResponse{
			Others: s.getClientIds(sc.cid),
		},
	}
	return sc.sendMessage(rsp)
}

// Handle an incoming Relay Request Message
// This one is trickier.
func (s *Server) handleRelayRequest(sc *serverClient, mesg *msg.Message) {

}

// Add a new client connection
func (s *Server) addClientByConnection(c net.Conn) {
	// Generate CID, add it to the map, start the dispatcher for it
	new_cid := msg.ClientId(atomic.AddUint64((*uint64)(&s.cid), 1))
	new_sc := serverClient{
		cid: new_cid,
		tc:  &msg.CborTranscoder{},
		dc:  msg.NewCborStreamDecoder(c),
		con: c,
	}
	s.clients_mutex.Lock()
	s.clients[new_cid] = new_sc
	s.clients_mutex.Unlock()
	s.startDispatcher(new_sc)
}

// Remove a client from server mapping
func (s *Server) removeClient(cid msg.ClientId) {

}

// Get a new slice of all client IDs, removing the ID of the caller
func (s *Server) getClientIds(except_cid msg.ClientId) []msg.ClientId {
	s.clients_mutex.RLock()
	cids := make([]msg.ClientId, len(s.clients)-1)
	i := 0
	for k := range s.clients {
		if k != except_cid {
			cids[i] = k
			i++
		}
	}
	s.clients_mutex.RUnlock()
	return cids
}

func (sc *serverClient) sendMessage(m msg.Message) msg.Status {
	encoded_msg, ok := sc.tc.Encode(m)
	if !ok {
		return msg.ENCODING_ERROR
	}
	n, err := sc.con.Write(encoded_msg)
	if (err != nil) || (n != len(encoded_msg)) {
		return msg.CONNECTION_ERROR
	}
	return msg.SUCCESS
}

/*
Package server implements the user-facing API of a broadcast_hub server.

Example, creating a listening TCP server on port 2593:
  ser := server.NewServer()
  listener, err := net.Listen("tcp", ":2593")
  if err == nil {
	  ser.AddListener(listener)
  }

*/
package server

import (
	"log"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/CiaranWoodward/broadcast_hub/msg"
)

// Maximum buffered messages per destination
const maxBufferedMessages = 3

// server representation of a connected client
type serverClient struct {
	// Client Id
	cid msg.ClientId
	// Relayed message stream (buffered)
	relayMsgs chan msg.RelayIndication
	// Response messages channel (non-buffered) (only for dispatcher to send to)
	responseMsgs chan msg.Message
	// Message stream decoder
	tc msg.Transcoder
	dc msg.StreamDecoder
	// Internal connection state
	con net.Conn
}

// Server class representing all of the state of a broadcast_hub server.
type Server struct {
	// Internal client ID counter (for unique IDs)
	cid msg.ClientId
	// Map of all connected clients
	clients       map[msg.ClientId]serverClient
	clients_mutex sync.RWMutex
	// Slice of all listeners
	listeners       []net.Listener
	listeners_mutex sync.Mutex
	// Shutdown tracker, preventing corrupted state during shutdown
	is_closed       bool
	is_closed_mutex sync.RWMutex
}

// Create a new server, that will act as a hub and allow connected clients to communicate.
// The server does nothing by itself, and must be either configured to accept new connections
// with the 'AddListener' function, or individual connections added with the 'AddClientByConnection'
// function.
func NewServer() *Server {
	return &Server{
		clients:   make(map[msg.ClientId]serverClient),
		listeners: make([]net.Listener, 0),
	}
}

// Add a listener which will accept new incoming connections from clients automatically.
// The server will handle closing the listener when it shuts down.
// 'ok' return value will be true unless server is closed
func (s *Server) AddListener(l net.Listener) (ok bool) {
	// Shutdown catch
	ok = true
	s.is_closed_mutex.RLock()
	defer s.is_closed_mutex.RUnlock()
	if s.is_closed {
		ok = false
		return
	}
	// Add listener to internal list
	s.listeners_mutex.Lock()
	s.listeners = append(s.listeners, l)
	s.listeners_mutex.Unlock()
	// Actual listening goroutine
	go func() {
		for {
			con, err := l.Accept()
			if err != nil {
				log.Printf("Error: %s\n", err.Error())
				break
			}
			s.AddClientByConnection(con)
		}
	}()
	return
}

// Add a new client connection. This is mainly for testing and allowing dual client-server programs.
// The server will handle closing the connection when it shuts down.
// 'ok' return value will be true unless server is closed
func (s *Server) AddClientByConnection(c net.Conn) (ok bool) {
	// Shutdown catch
	ok = true
	s.is_closed_mutex.RLock()
	defer s.is_closed_mutex.RUnlock()
	if s.is_closed {
		ok = false
		return
	}
	// Generate CID, add it to the map, start the dispatcher for it
	new_cid := msg.ClientId(atomic.AddUint64((*uint64)(&s.cid), 1))
	tc := &msg.CborTranscoder{}
	new_sc := serverClient{
		cid:          new_cid,
		relayMsgs:    make(chan msg.RelayIndication, maxBufferedMessages),
		responseMsgs: make(chan msg.Message),
		tc:           tc,
		dc:           tc.NewStreamDecoder(c),
		con:          c,
	}
	s.clients_mutex.Lock()
	s.clients[new_cid] = new_sc
	s.clients_mutex.Unlock()
	s.startDispatcher(new_sc)
	s.startSender(new_sc)
	log.Printf("Added new Client %d\n", new_cid)
	return
}

// Close the server, and all associated resources and connections
func (s *Server) Close() {
	// Disable all public functions
	s.is_closed_mutex.Lock()
	defer s.is_closed_mutex.Unlock()
	s.is_closed = true
	// Close all listeners and clients
	s.closeAllListeners()
	s.closeAllClients()
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
				break
			}
		}
		// Close connection - this will trigger sender to shut down and clean up
		sc.con.Close()
		close(sc.responseMsgs)
	}()
}

func (s *Server) startSender(sc serverClient) {
	// Write messages to the transport, prioritising responses over relayed messages
	go func() {
		// Counter for unique MIDs in indications
		relay_mid := uint32(0)
		for {
			mesg := msg.Message{}
			// Nested select for prioritization.
			select {
			case mesg = <-sc.responseMsgs:
			default:
				select {
				case mesg = <-sc.responseMsgs:
				case relayed := <-sc.relayMsgs:
					mesg.Version = msg.MyVersion
					mesg.MessageId = relay_mid
					mesg.RelayInd = &relayed
					relay_mid++
				}
			}
			// Actually send the message
			if sc.sendMessage(mesg) == msg.CONNECTION_ERROR {
				break
			}
		}
		// Cleanup
		s.removeClient(sc.cid)
		// Wait for dispatcher to shut down
	shutdown_loop:
		for {
			select {
			case _, ok := <-sc.responseMsgs:
				if !ok {
					break shutdown_loop
				}
			case <-time.After(time.Second * 15):
				panic("Failed to clean up serverClient!")
			}
		}
		log.Printf("Removed Client %d\n", sc.cid)
	}()
}

// Handle an incoming ID Request Message
func (s *Server) handleIdRequest(sc *serverClient, mesg *msg.Message) {
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		IdRes: &msg.IdentifyResponse{
			Id: sc.cid,
		},
	}
	sc.responseMsgs <- rsp
}

// Handle an incoming List Request Message (This has a potentially unbounded response size. Limited by number of connected clients.)
func (s *Server) handleListRequest(sc *serverClient, mesg *msg.Message) {
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		ListRes: &msg.ListResponse{
			Others: s.getClientIds(sc.cid),
		},
	}
	sc.responseMsgs <- rsp
}

// Handle an incoming Relay Request Message
func (s *Server) handleRelayRequest(sc *serverClient, mesg *msg.Message) {
	// Iterate through all clients' buffered channels, and send the message to each of them,
	// if it can be done without blocking. Otherwise, fail with NO_BUFFER.
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		RelayRes: &msg.RelayResponse{
			Status:    msg.SUCCESS,
			StatusMap: make(msg.ClientStatusMap),
		},
	}
	if len(mesg.RelayReq.Dest) > 255 || len(mesg.RelayReq.Msg) > 1024 {
		rsp.RelayRes.Status = msg.TOO_LONG
	} else {
		rsp.RelayRes.StatusMap = s.sendRelays(sc, mesg)
	}
	sc.responseMsgs <- rsp
}

// Handle forwarding the relay messages to each individual destination
func (s *Server) sendRelays(sc *serverClient, request *msg.Message) msg.ClientStatusMap {
	statusMap := make(msg.ClientStatusMap)
	ind := msg.RelayIndication{
		Src: sc.cid,
		Msg: request.RelayReq.Msg,
	}
	for _, cid := range request.RelayReq.Dest {
		s.clients_mutex.RLock()
		dest_client, ok := s.clients[cid]
		if !ok {
			statusMap[cid] = msg.INVALID_ID
			s.clients_mutex.RUnlock()
			continue
		}
		dest_chan := dest_client.relayMsgs
		s.clients_mutex.RUnlock()

		//Nonblocking send to buffered channel
		select {
		case dest_chan <- ind:
			// Success! (We don't report successes in the response)
			// The client will receive the relay indication soon, unless it disconnects first. (best effort relay)
			// TODO: Do we want a better delivery guarantee?
		default:
			statusMap[cid] = msg.NO_BUFFER
			continue
		}
	}
	return statusMap
}

// Close all listeners
func (s *Server) closeAllListeners() {
	s.listeners_mutex.Lock()
	for _, l := range s.listeners {
		l.Close()
	}
	s.listeners_mutex.Unlock()
}

// Close all clients
func (s *Server) closeAllClients() {
	s.clients_mutex.RLock()
	for _, cli := range s.clients {
		cli.con.Close()
	}
	s.clients_mutex.RUnlock()
}

// Remove a client from server mapping, and close its connection.
// This should only be called by the sender goroutine.
func (s *Server) removeClient(cid msg.ClientId) {
	s.clients_mutex.Lock()
	cli, ok := s.clients[cid]
	if ok {
		cli.con.Close()
	}
	delete(s.clients, cid)
	s.clients_mutex.Unlock()
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

// Encode and send a message over the transport to the client
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

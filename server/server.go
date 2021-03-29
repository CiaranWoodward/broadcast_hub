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

// Maximum buffered messages per destination
const maxBufferedMessages = 3

// server representation of a connected client
type serverClient struct {
	// Client Id
	cid msg.ClientId
	// Relayed message stream (buffered)
	relayMsgs chan msg.RelayIndication
	// Response message stream (non-buffered)
	responseMsgs chan msg.Message
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

func (s *Server) startSender(sc serverClient) {
	// Write messages to the transport, prioritising responses over relayed messages
	go func() {
		relay_mid := uint32(0)
		for {
			mesg := msg.Message{}
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
	/* Design thoughts:
	 * 3 options come to mind:
	 * 1. Iterate through all clients and write to all of them (No, far too io-bound)
	 *
	 * 2. Iterate through all clients and write to each of them in a new goroutine
	 *    (No, boundless goroutine creation, difficult to throttle per destination)
	 *
	 * 3. Iterate through all clients' buffered channels, and send the message to each of them,
	 *    if it can be done without blocking. Otherwise, fail.
	 *    Slightly suboptimal use of network in simple case, but can trivially be enhanced
	 *    to optimise latency & throughput (by running a second dispatcher goroutine per client
	 *    if its important)
	 */
	rsp := msg.Message{
		Version:   msg.MyVersion,
		MessageId: mesg.MessageId,
		RelayRes: &msg.RelayResponse{
			Status:    msg.SUCCESS,
			StatusMap: make(msg.ClientStatusMap),
		},
	}
	ind := msg.RelayIndication{
		Src: sc.cid,
		Msg: mesg.RelayReq.Msg,
	}
	for _, cid := range mesg.RelayReq.Dest {
		s.clients_mutex.RLock()
		dest_client, ok := s.clients[cid]
		if !ok {
			rsp.RelayRes.StatusMap[cid] = msg.INVALID_ID
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
			rsp.RelayRes.StatusMap[cid] = msg.NO_BUFFER
			continue
		}
	}
}

// Add a new client connection
func (s *Server) addClientByConnection(c net.Conn) {
	// Generate CID, add it to the map, start the dispatcher for it
	new_cid := msg.ClientId(atomic.AddUint64((*uint64)(&s.cid), 1))
	new_sc := serverClient{
		cid:          new_cid,
		relayMsgs:    make(chan msg.RelayIndication, maxBufferedMessages),
		responseMsgs: make(chan msg.Message),
		tc:           &msg.CborTranscoder{},
		dc:           msg.NewCborStreamDecoder(c),
		con:          c,
	}
	s.clients_mutex.Lock()
	s.clients[new_cid] = new_sc
	s.clients_mutex.Unlock()
	s.startDispatcher(new_sc)
	s.startSender(new_sc)
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

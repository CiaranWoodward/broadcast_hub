/*
All of the definitions supported by the bhub protocol.

Every message contains:
 - "bhub-ver" = 1
   - Unique 8 byte string for protocol identification
   - Version can be incremented for future versions
 - Message ID
   - Unique per command-response pair
   - Links response messages to requests (same ID)
 - Map containing the actual command type
   - The underlying message structure supports combining multiple commands per message, but this is not currently used in the protocol.
 - Additional fields as the 'map' values based on command ID

Terminology:
 "Request" For an unsolicited message from Client to Hub.
 "Response" For a reply to a "Request"
 "Indication" For an unsolicited message from Hub to Client

Commands (with direction):
 C = Client
 H = Hub (Server)
 - Identify Request (C->H)
 - Identify Response (C<-H)
    - Id: ClientId
 - List Request (C->H)
 - List Response (H<-C)
    - Others: Array of ClientIds
 - Relay Request (C->H)
    - Dest: Array of ClientIds
    - Message: Byte array
 - Relay Response (C<-H)
    - Array of (ClientId, Status) tuples
 - Relay Indication (C<-H)
    - Source: ClientId
    - Message: Byte array
*/
package msg

import (
	"fmt"
	"io"
)

// ClientId type, unique id per client
type ClientId uint64

// Status value, including success
type Status int

const (
	// Successful status
	SUCCESS Status = iota
	// ID Not valid
	INVALID_ID
	// No buffer space remaining
	NO_BUFFER
	// Underlying connection error
	CONNECTION_ERROR
	// Message encoding error
	ENCODING_ERROR
	// Protocol timed out before response was received
	TIMEOUT
	// One of the parameters is longer than the protocol allows
	TOO_LONG
)

// Version type, only version 1 currently supported
type Version int

const MyVersion Version = 1

// ClientStatusMap is a map of clientIDs to their respective status
type ClientStatusMap map[ClientId]Status

// Message is the message that is actually sent over the transport, with
// subfields to represent all of the other message types.
type Message struct {
	Version   Version           `json:"bhubver"`
	MessageId uint32            `json:"id"`
	IdReq     *IdentifyRequest  `json:"ir,omitempty"`
	IdRes     *IdentifyResponse `json:"IR,omitempty"`
	ListReq   *ListRequest      `json:"lr,omitempty"`
	ListRes   *ListResponse     `json:"LR,omitempty"`
	RelayReq  *RelayRequest     `json:"rr,omitempty"`
	RelayRes  *RelayResponse    `json:"RR,omitempty"`
	RelayInd  *RelayIndication  `json:"RI,omitempty"`
}

// IdentifyRequest is a identify message request from Client to Hub to get its client ID
type IdentifyRequest struct {
}

// IdentifyResponse is the response to the IdentifyRequest, identifying the client
type IdentifyResponse struct {
	Id ClientId `json:"id"`
}

// ListRequest is a request from client to hub to list all other client IDs connected to the hub
type ListRequest struct {
}

// ListResponse is the response to ListRequest, listing all other connected Clients by ID
type ListResponse struct {
	Others []ClientId `json:"o"`
}

// RelayRequest is a request from client to hub to request a message to be relayed to a list of other clients
type RelayRequest struct {
	Dest []ClientId `json:"dst"`
	Msg  []byte     `json:"msg"`
}

// RelayResponse is the response to RelayRequest, containing a status for each client the message was relayed to
// There is also an overall status field, for the case where the message was not relayed at all.
// The StatusMap does not include successes, so if a Client ID is not present, it can be assumed to be successful.
type RelayResponse struct {
	Status    Status          `json:"sta"`
	StatusMap ClientStatusMap `json:"csm"`
}

// RelayIndication is a message from the hub to a client, containing the source of the message, and the message itself
type RelayIndication struct {
	Src ClientId `json:"src"`
	Msg []byte   `json:"msg"`
}

// The transcoder interface serializes/deserializes messages to byte arrays.
// This allows for flexibility in message format for development/testing, and decouples the message format from the transport
type Transcoder interface {
	Encode(msgin Message) (msgout []byte, ok bool)
	Decode(msgin []byte) (msgout Message, ok bool)
	NewStreamDecoder(r io.Reader) StreamDecoder
}

// The StreamDecoder decodes and de-packetises messages from a stream
type StreamDecoder interface {
	DecodeNext() (msgout Message, ok bool)
}

func (s Status) String() string {
	switch s {
	case SUCCESS:
		return "SUCCESS"
	case INVALID_ID:
		return "INVALID_ID"
	case NO_BUFFER:
		return "NO_BUFFER"
	case CONNECTION_ERROR:
		return "CONNECTION_ERROR"
	case ENCODING_ERROR:
		return "ENCODING_ERROR"
	case TIMEOUT:
		return "TIMEOUT"
	case TOO_LONG:
		return "TOO_LONG"
	default:
		return fmt.Sprintf("[Unknown Status: %d]", int(s))
	}
}

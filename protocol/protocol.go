/*
All of the definitions supported by the bhub protocol

Every message contains:
 - Map of "bhub-ver" : 1
   - Can be incremented for future versions
 - Message ID
   - Unique per command-response pair
   - Links response messages to requests (same ID)
 - Map containing the actual command type
   - The underlying message structure supports combining multiple commands per message, but this is not currently used.
 - optional additional fields based on command ID

Commands:
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
package protocol

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
)

// Version type, only version 1 currently supported
type Version int

const MyVersion Version = 1

// ClientStatusMap is a map of clientIDs to their respective status
type ClientStatusMap map[ClientId]Status

// Message struct is the highest level of message that are sent over the transport
type Message struct {
	Version   Version           `json:"bhubver"`
	MessageId int32             `json:"id"`
	IdReq     *IdentifyRequest  `json:"ir,omitempty"`
	IdRes     *IdentifyResponse `json:"IR,omitempty"`
	ListReq   *ListRequest      `json:"lr,omitempty"`
	ListRes   *ListResponse     `json:"LR,omitempty"`
	RelayReq  *RelayRequest     `json:"rr,omitempty"`
	RelayRes  *RelayResponse    `json:"RR,omitempty"`
	RelayInd  *RelayIndication  `json:"RI,omitempty"`
}

// IdentifyRequest is a self-identify message request from Client to Hub
type IdentifyRequest struct {
}

// IdentifyResponse is the response to the IdentifyRequest, identifying the client
type IdentifyResponse struct {
	Id ClientId `json:"id"`
}

// ListRequest is a request from client to hub to identify all other client IDs connected to the hub
type ListRequest struct {
}

// ListResponse is the response to ListRequest, listing all other connected Clients by ID
type ListResponse struct {
	Others []ClientId `json:"o"`
}

// RelayRequest is a request from client to hub to request a message to be relayed to all other clients
type RelayRequest struct {
	Dest []ClientId `json:"dst"`
	Msg  []byte     `json:"msg"`
}

// RelayResponse is the response to RelayRequest, containing a status for each client the message was relayed to
// There is also an overall status field, for the case where the message was not relayed at all
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
}

// Need to be able to decode messages from stream (stream encoding is not so necessary)
type Decoder interface {
	Decode() (msgout Message, ok bool)
}

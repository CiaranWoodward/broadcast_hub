/*
All of the definitions supported by the bhub protocol

Every message contains:
 - Map of "bhub-ver" : 1
   - Can be incremented for future versions
 - Command ID
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
)

// CommandId type, each is uique per command
type CommandId int

const (
	IdIdentifyRequest  CommandId = 1
	IdIdentifyResponse CommandId = 2
	IdListRequest      CommandId = 3
	IdListResponse     CommandId = 4
	IdRelayRequest     CommandId = 5
	IdRelayResponse    CommandId = 6
	IdRelayIndication  CommandId = 7
)

// Command Interface, must be implemented by all command types
type Command interface{}

// Version type, only version 1 currently supported
type Version int

// ClientStatusMap is a map of clientIDs to their respective status
type ClientStatusMap map[ClientId]Status

type Header struct {
	bhubVer Version
	msgId   CommandId
}

type Message struct {
	header  Header
	command *Command
}

// IdentifyRequest is a self-identify message request from Client to Hub
type IdentifyRequest struct {
}

// IdentifyResponse is the response to the IdentifyRequest, identifying the client
type IdentifyResponse struct {
	id ClientId
}

// ListRequest is a request from client to hub to identify all other client IDs connected to the hub
type ListRequest struct {
}

// ListResponse is the response to ListRequest, listing all other connected Clients by ID
type ListResponse struct {
	others []ClientId
}

// RelayRequest is a request from client to hub to request a message to be relayed to all other clients
type RelayRequest struct {
	dest []ClientId
	msg  []byte
}

// RelayResponse is the response to RelayRequest, containing a status for each client the message was relayed to
type RelayResponse struct {
	status ClientStatusMap
}

// RelayIndication is a message from the hub to a client, containing the source of the message, and the message itself
type RelayIndication struct {
	src ClientId
	msg []byte
}

// The transcoder interface serializes/deserializes messages to byte arrays.
// This allows for flexibility in message format for development/testing, and decouples the message format from the transport
type Transcoder interface {
	Encode(msgin Message) (msgout []byte, ok bool)
	Decode(msgin []byte) (msgout Message, ok bool)
}

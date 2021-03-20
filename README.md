# Broadcast hub

Simple toy client-server protocol allowing multiple 'clients' to connect to a 'hub' server, and use the hub
to relay messages to multiple destinations.

This should be written in a manner that is easily testible, benchmark-able, and ideally in a manner than allows
swapping out components like the transport mechanism. The server should be very failure-tolerant.

Initial protocol architecture:
 - Messages represented using [CBOR](https://tools.ietf.org/html/rfc8949)
    - All messages should include the protocol version and something identifying the message as part of the 'bhub' protocol
    - Each message should include the command type, so that the server can identify it.
 - Initial transport is using TCP (Will later upgrade to TLS)
 - Have some mechanism to swap out the TCP for some mock-able local 'transport'

Messages:
 - Identify Request
 - Identify Response
    - Id: ClientId
 - List Request
 - List Response
    - Others: Array of ClientIds
 - Relay Request
    - Dest: Array of ClientIds
    - Message: Byte array
 - Relay Response
    - Array of (ClientId, Status) tuples
 - Relayed Msg
    - Source: ClientId
    - Message: Byte array
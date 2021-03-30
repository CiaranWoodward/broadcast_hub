# Broadcast hub

Simple toy client-server protocol allowing multiple 'clients' to connect to a 'hub' server, and use the hub
to relay messages to multiple destinations.

This is written in a way that is easily testable, benchmark-able, and in a manner than allows
swapping out components like the transport mechanism. The server should be very failure-tolerant.

There is a comprehensive test suite included, testing message structure, client standalone and full client + server systems.
The tests make use of a tool which checks for goroutine leakage.

Protocol architecture:
 - Messages represented using [CBOR](https://tools.ietf.org/html/rfc8949)
    - All messages include the protocol version and and a protocol-identifying string
    - Each message contains a label identifying which command it is (a map key)
    - There is also a debug encoder included, which uses JSON instead, for human readability.
 - Protocol is fairly transport-agnostic
    - Currently TCP is used
    - TLS would be an easy upgrade, as makes use of Go's "net.Conn" interface 
    - In tests, the even simpler 'net.Pipe' is used

Terminology:
 - "Request" For an unsolicited message from Client to Hub.
 - "Response" For a reply to a "Request"
 - "Indication" For an unsolicited message from Hub to Client

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
    - Status: Status
    - Array of (ClientId, Status) tuples for individual failures
 - Relay Indication (C<-H)
    - Source: ClientId
    - Message: Byte array

## Directory layout

 - ``msg``    Contains the core protocol message structure, data types & transcoders
 - ``client`` Contains all of the source and tests for the broadcast_hub client
 - ``server`` Contains all of the source and tests for the broadcast_hub server
 - ``cmd``    Contains the example CLI applications for hand-testing

## Testing

Tests can be run using the ``go test`` tool:

(The msg module is 100% coverage if you ignore an enum->string conversion function)
```
PS D:\Working\go\broadcast_hub> go test -cover  ./...
ok      github.com/CiaranWoodward/broadcast_hub/client  5.068s  coverage: 80.2% of statements
?       github.com/CiaranWoodward/broadcast_hub/cmd/bhclient    [no test files]
?       github.com/CiaranWoodward/broadcast_hub/cmd/bhserver    [no test files]
ok      github.com/CiaranWoodward/broadcast_hub/msg     0.034s  coverage: 69.0% of statements
ok      github.com/CiaranWoodward/broadcast_hub/server  8.129s  coverage: 94.4% of statements
```

## Building/Installing

The commands can be built/run locally by running ``go build`` in the individual command directories.

Or the entire project can be installed with ``go install ./...`` in the root directory.

## Running Demo Server

The server takes a single ``-p`` option, designating the TCP port it will bind to.

```
D:\Working\go\broadcast_hub\cmd\bhserver> .\bhserver.exe -p 3030
2021/03/29 23:01:18 Successfully listening on port 3030.
2021/03/29 23:01:18 Use Ctl-C to exit.
2021/03/29 23:01:23 Added new Client 1
2021/03/29 23:01:23 Added new Client 2
2021/03/29 23:01:23 Added new Client 3
2021/03/29 23:01:23 Added new Client 4
```

## Running Demo Client

The client has more extensive options:

```
$ bhclient.exe -h
NAME:
   client - The broadcast_hub client, for connecting to a server and communicating with other clients

USAGE:
   bhclient.exe [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --server HOSTNAME, -s HOSTNAME  Connect to the broadcast_hub server at the provided HOSTNAME.
   --port PORT, -p PORT            Connect to the given PORT of the broadcast_hub server. (default: 0)
   --roger_no COUNT                Create the given COUNT of dummy clients, which will respond back with a message whenever they are contacted (default: 0)
   --help, -h                      show help (default: false)
```

In particular, the ``roger_no`` option can be used to create a large number of clients which reply back to messages, for test purposes.

Once in the client, there is a simple console that allows sending commands.

Example:
```
PS D:\Working\go\broadcast_hub\cmd> bhclient.exe -s localhost -p 3030 --roger_no 50
Successfully connected to server localhost:3030, with CID 18361.
Successfully started Roger 18362
Interactive Help:
 getid
    - Get the ID of this client
 list
    - Get the IDs of the other connected clients
 relay <space separated list of Client IDs> : <ASCII Message>
    - Send a message to the list of other Clients, via the hub.
      Eg: relay 1 2 34 :Hello there!
 quit
Successfully started Roger 18363
Successfully started Roger 18365
Successfully started Roger 18364
 <snip>
Successfully started Roger 18410
Successfully started Roger 18411
>
>getid
My ID: 18361
>
>list
Other IDs: [18378 18402 18385 18388 18393 18382 18391 18377 18397 18411 18407 18379 18364 18387 18390 18383 18365 18362 18366 18410 18380 18389 18403 18384 18406 18372 18369 18381 18373 18376 18399 18400 18409 18363 18404 18367 18386 18370 18392 18405 18394 18374 18368 18395 18398 18408 18375 18401 18371 18396]
>
>relay 18378 18402 18385:Hello Roger!
2021/03/30 03:39:46 Success!
>Rx from 18385: Roger that 18361 - I am 18385!
Rx from 18402: Roger that 18361 - I am 18402!
Rx from 18378: Roger that 18361 - I am 18378!
```

## Future Work

- Swap TCP for TLS
   - Security is important
   - Can also experiment with other technologies such as websockets.
   - UDP not immediately suitable
 - Improve server throttling of clients
   - Currently we throttle to avoid overloading destinations, but don't throttle aggressive senders
 - More testing, including stress testing and trying to break the server
 - Persistent Client IDs (by session token allocation or something, but must be over secure connection)
 - Improve 'fairness' of which messages will actually get through to a slow client
   - Currently 'responses' are prioritized, but an aggressive fast client can still mostly prevent a slow client from receiving other relay indications.

And at the protocol level:
 - The List message limits scalability. To be useful, it would need to be replaced by some mechanism of sending to groups instead of having to query ALL individuals.
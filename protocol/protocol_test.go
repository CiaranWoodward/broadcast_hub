package protocol

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

type cborTestElement struct {
	name      string
	msg       Message
	hexString string
}

var cborTestVec = []cborTestElement{
	{
		"Identify Request",
		Message{Version: MyVersion, MessageId: 0x12, IdReq: &IdentifyRequest{}},
		"a367626875627665720162696412626972a0",
	},
	{
		"Identify Response",
		Message{Version: MyVersion, MessageId: 0x34, IdRes: &IdentifyResponse{1234}},
		"a36762687562766572016269641834624952a16269641904d2",
	},
	{
		"List Request",
		Message{Version: MyVersion, MessageId: 0x56, ListReq: &ListRequest{}},
		"a36762687562766572016269641856626c72a0",
	},
	{
		"List Response",
		Message{Version: MyVersion, MessageId: 0x78, ListRes: &ListResponse{Others: []ClientId{1, 2, 3, 0xFFFFFFFFFFFFFFFF}}},
		"a36762687562766572016269641878624c52a1616f840102031bffffffffffffffff",
	},
	{
		"Relay Request",
		Message{Version: MyVersion, MessageId: 0x9A, RelayReq: &RelayRequest{Dest: []ClientId{1, 2, 3}, Msg: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}}},
		"a3676268756276657201626964189a627272a26364737483010203636d7367460123456789ab",
	},
	{
		"Relay Response",
		Message{Version: MyVersion, MessageId: 0xBC, RelayRes: &RelayResponse{Status: SUCCESS, StatusMap: ClientStatusMap{2: NO_BUFFER, 3: INVALID_ID}}},
		"", // No byte comparison as status map is unordered
	},
	{
		"Relay Request",
		Message{Version: MyVersion, MessageId: 0xDE, RelayInd: &RelayIndication{Src: 1234, Msg: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}}},
		"a367626875627665720162696418de625249a2637372631904d2636d7367460123456789ab",
	},
}

// Simple CBOR loopback test to check everything can be decoded from its encoded form
func TestCborEncoder(t *testing.T) {
	tc := CborTranscoder{}
	for _, testElem := range cborTestVec {
		t.Run(testElem.name, func(t *testing.T) {
			// Encode a command message
			msg := testElem.msg
			encoded, ok := tc.Encode(msg)
			assert.True(t, ok)

			// Assert it is the correct byte sequence (If byte sequence is included in test vec)
			fmt.Println(hex.EncodeToString(encoded))
			if testElem.hexString != "" {
				exBytes, _ := hex.DecodeString(testElem.hexString)
				assert.Equal(t, encoded, exBytes)
			}

			// Loop it back, and confirm it is the same as before
			msgOut, ok := tc.Decode(encoded)
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut)

			// And also with the stream decoder
			sd := NewCborDecoder(bytes.NewReader(encoded))
			msgOut2, ok := sd.Decode()
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut2)
		})
	}
}

// Simple JSON loopback test to check everything can be decoded from its encoded form
// This is based off the CBOR test vector, just doesn't check the encoded form matches
// the expected binary value, as json is less predictable and is only included here for
// debugging.
func TestJsonEncoder(t *testing.T) {
	tc := JsonTranscoder{}
	for _, testElem := range cborTestVec {
		t.Run(testElem.name, func(t *testing.T) {
			// Encode a command message
			msg := testElem.msg
			encoded, ok := tc.Encode(msg)
			assert.True(t, ok)

			fmt.Println(string(encoded))

			// Loop it back, and confirm it is the same as before
			msgOut, ok := tc.Decode(encoded)
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut)

			// And also with the stream decoder
			sd := NewJsonDecoder(bytes.NewReader(encoded))
			msgOut2, ok := sd.Decode()
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut2)
		})
	}
}

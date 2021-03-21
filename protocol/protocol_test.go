package protocol

import (
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
		Message{Version: MyVersion, IdReq: &IdentifyRequest{}},
		"a2676268756276657201626972a0",
	},
	{
		"Identify Response",
		Message{Version: MyVersion, IdRes: &IdentifyResponse{1234}},
		"a2676268756276657201624952a16269641904d2",
	},
	{
		"List Request",
		Message{Version: MyVersion, ListReq: &ListRequest{}},
		"a2676268756276657201626c72a0",
	},
	{
		"List Response",
		Message{Version: MyVersion, ListRes: &ListResponse{Others: []ClientId{1, 2, 3, 0xFFFFFFFFFFFFFFFF}}},
		"a2676268756276657201624c52a1616f840102031bffffffffffffffff",
	},
	{
		"Relay Request",
		Message{Version: MyVersion, RelayReq: &RelayRequest{Dest: []ClientId{1, 2, 3}, Msg: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}}},
		"a2676268756276657201627272a26364737483010203636d7367460123456789ab",
	},
	{
		"Relay Response",
		Message{Version: MyVersion, RelayRes: &RelayResponse{Status: ClientStatusMap{1: SUCCESS, 2: NO_BUFFER, 3: INVALID_ID}}},
		"a2676268756276657201625252a16363736da3010002020301",
	},
	{
		"Relay Request",
		Message{Version: MyVersion, RelayInd: &RelayIndication{Src: 1234, Msg: []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB}}},
		"a2676268756276657201625249a2637372631904d2636d7367460123456789ab",
	},
}

// Simple loopback test to check everything can be decoded from its encoded form
func TestCborEncoder(t *testing.T) {
	tc := CborTranscoder{}
	for _, testElem := range cborTestVec {
		t.Run(testElem.name, func(t *testing.T) {
			// Encode a command message
			msg := testElem.msg
			bytes, ok := tc.Encode(msg)
			assert.True(t, ok)

			// Assert it is the correct byte sequence (If byte sequence is included in test vec)
			fmt.Println(hex.EncodeToString(bytes))
			if testElem.hexString != "" {
				exBytes, _ := hex.DecodeString(testElem.hexString)
				assert.Equal(t, bytes, exBytes)
			}

			// Loop it back, and confirm it is the same as before
			msgOut, ok := tc.Decode(bytes)
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut)
		})
	}
}

// Simple loopback test to check everything can be decoded from its encoded form
// This is based off the CBOR test vector, just doens't check the encoded form matches
// the expected binary value, as json is less predictable and is only included here for
// debugging.
func TestJsonEncoder(t *testing.T) {
	tc := JsonTranscoder{}
	for _, testElem := range cborTestVec {
		t.Run(testElem.name, func(t *testing.T) {
			// Encode a command message
			msg := testElem.msg
			bytes, ok := tc.Encode(msg)
			assert.True(t, ok)

			// Loop it back, and confirm it is the same as before
			msgOut, ok := tc.Decode(bytes)
			assert.True(t, ok)
			assert.Equal(t, testElem.msg, msgOut)
		})
	}
}

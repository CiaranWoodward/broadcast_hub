package protocol

import "github.com/fxamacker/cbor/v2"

type CborTranscoder struct {
}

func (*CborTranscoder) Encode(msgin Message) (msgout []byte, ok bool) {
	msgout, err := cbor.Marshal(msgin)
	ok = (err == nil)
	return
}

func (*CborTranscoder) Decode(msgin []byte) (msgout Message, ok bool) {
	err := cbor.Unmarshal(msgin, &msgout)
	ok = (err == nil)
	return
}

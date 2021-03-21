package protocol

import (
	"io"

	"github.com/fxamacker/cbor/v2"
)

type CborTranscoder struct {
}

type CborDecoder struct {
	dec *cbor.Decoder
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

func NewCborDecoder(r io.Reader) *CborDecoder {
	return &CborDecoder{dec: cbor.NewDecoder(r)}
}

func (cd *CborDecoder) Decode() (msgout Message, ok bool) {
	err := cd.dec.Decode(&msgout)
	ok = (err == nil)
	return
}

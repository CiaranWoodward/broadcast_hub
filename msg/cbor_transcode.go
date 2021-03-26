package msg

import (
	"io"

	"github.com/fxamacker/cbor/v2"
)

type CborTranscoder struct {
}

type CborStreamDecoder struct {
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

func NewCborStreamDecoder(r io.Reader) *CborStreamDecoder {
	return &CborStreamDecoder{dec: cbor.NewDecoder(r)}
}

func (cd *CborStreamDecoder) DecodeNext() (msgout Message, ok bool) {
	err := cd.dec.Decode(&msgout)
	ok = (err == nil)
	return
}

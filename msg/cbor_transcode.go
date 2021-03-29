package msg

import (
	"io"

	"github.com/fxamacker/cbor/v2"
)

// CBOR Implementation of the Transcoder interface
type CborTranscoder struct {
}

type cborStreamDecoder struct {
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

func (*CborTranscoder) NewStreamDecoder(r io.Reader) StreamDecoder {
	return &cborStreamDecoder{dec: cbor.NewDecoder(r)}
}

func (cd *cborStreamDecoder) DecodeNext() (msgout Message, ok bool) {
	err := cd.dec.Decode(&msgout)
	ok = (err == nil)
	return
}

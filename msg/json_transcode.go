package msg

import (
	"encoding/json"
	"io"
)

// JSON Implementation of the Transcoder interface
type JsonTranscoder struct {
}

type jsonDecoder struct {
	dec *json.Decoder
}

func (*JsonTranscoder) Encode(msgin Message) (msgout []byte, ok bool) {
	msgout, err := json.Marshal(msgin)
	ok = (err == nil)
	return
}

func (*JsonTranscoder) Decode(msgin []byte) (msgout Message, ok bool) {
	err := json.Unmarshal(msgin, &msgout)
	ok = (err == nil)
	return
}

func (*JsonTranscoder) NewStreamDecoder(r io.Reader) StreamDecoder {
	return &jsonDecoder{dec: json.NewDecoder(r)}
}

func (jd *jsonDecoder) DecodeNext() (msgout Message, ok bool) {
	err := jd.dec.Decode(&msgout)
	ok = (err == nil)
	return
}

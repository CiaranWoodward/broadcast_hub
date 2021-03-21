package protocol

import (
	"encoding/json"
	"io"
)

type JsonTranscoder struct {
}

type JsonDecoder struct {
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

func NewJsonDecoder(r io.Reader) *JsonDecoder {
	return &JsonDecoder{dec: json.NewDecoder(r)}
}

func (jd *JsonDecoder) Decode() (msgout Message, ok bool) {
	err := jd.dec.Decode(&msgout)
	ok = (err == nil)
	return
}

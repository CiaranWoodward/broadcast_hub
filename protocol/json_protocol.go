package protocol

import "encoding/json"

type JsonTranscoder struct {
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

package client

import (
	"net"
	"testing"

	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/stretchr/testify/assert"
)

func TestClientIdReq(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to receive ID request, verify it, and send a response
	go func() {
		sd := msg.NewCborStreamDecoder(ser)
		en := msg.CborTranscoder{}
		m, ok := sd.DecodeNext()
		assert.True(t, ok)
		assert.Equal(t, msg.MyVersion, m.Version)
		assert.NotNil(t, m.IdReq)
		assert.Nil(t, m.IdRes)
		assert.Nil(t, m.ListReq)
		assert.Nil(t, m.ListRes)
		assert.Nil(t, m.RelayReq)
		assert.Nil(t, m.RelayRes)
		assert.Nil(t, m.RelayInd)
		rsp := msg.Message{
			Version:   msg.MyVersion,
			MessageId: m.MessageId,
			IdRes:     &msg.IdentifyResponse{Id: 1234},
		}
		rspb, ok := en.Encode(rsp)
		assert.True(t, ok)
		n, err := ser.Write(rspb)
		assert.Equal(t, len(rspb), n)
		assert.Nil(t, err)
	}()

	tc := NewClient(cli)
	cid, status := tc.GetClientId()
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, msg.ClientId(1234), cid)
}

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

func TestClientListReq(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to receive ID request, verify it, and send a response
	go func() {
		sd := msg.NewCborStreamDecoder(ser)
		en := msg.CborTranscoder{}
		m, ok := sd.DecodeNext()
		assert.True(t, ok)
		assert.Equal(t, msg.MyVersion, m.Version)
		assert.Nil(t, m.IdReq)
		assert.Nil(t, m.IdRes)
		assert.NotNil(t, m.ListReq)
		assert.Nil(t, m.ListRes)
		assert.Nil(t, m.RelayReq)
		assert.Nil(t, m.RelayRes)
		assert.Nil(t, m.RelayInd)
		rsp := msg.Message{
			Version:   msg.MyVersion,
			MessageId: m.MessageId,
			ListRes:   &msg.ListResponse{Others: []msg.ClientId{1, 2, 3, 4, 5}},
		}
		rspb, ok := en.Encode(rsp)
		assert.True(t, ok)
		n, err := ser.Write(rspb)
		assert.Equal(t, len(rspb), n)
		assert.Nil(t, err)
	}()

	tc := NewClient(cli)
	cids, status := tc.ListOtherClients()
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, []msg.ClientId{1, 2, 3, 4, 5}, cids)
}

func TestClientIdConnBreak(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to receive ID request, then terminate connection
	go func() {
		sd := msg.NewCborStreamDecoder(ser)
		sd.DecodeNext()
		// We received the message, terminate the connection while the client is waiting for response!
		ser.Close()
	}()

	tc := NewClient(cli)
	_, status := tc.GetClientId()
	assert.Equal(t, msg.CONNECTION_ERROR, status)
}

func TestClientIdTimeout(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to receive ID request, but not respond
	go func() {
		sd := msg.NewCborStreamDecoder(ser)
		sd.DecodeNext()
	}()

	tc := NewClient(cli)
	_, status := tc.GetClientId()
	assert.Equal(t, msg.TIMEOUT, status)
}

func TestClientIdCloseMid(t *testing.T) {
	cli, ser := net.Pipe()

	tc := NewClient(cli)
	// Goroutine to close the client while it's mid sending the request (after 1 byte has been received)
	go func() {
		rcbuf := make([]byte, 1)
		n, err := ser.Read(rcbuf)
		assert.Nil(t, err)
		assert.Equal(t, 1, n)
		tc.Close()
	}()
	_, status := tc.GetClientId()
	assert.Equal(t, msg.CONNECTION_ERROR, status)
}

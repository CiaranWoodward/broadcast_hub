package client

import (
	"net"
	"testing"

	"github.com/CiaranWoodward/broadcast_hub/msg"
	"github.com/stretchr/testify/assert"
)

//TODO: Add a test to check that multiple MIDs work correctly (responses go to correct responders)

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

	// Fake server to receive List request, verify it, and send a response
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

func TestClientRelayReq(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to receive Relay request, verify it, and send a response
	go func() {
		sd := msg.NewCborStreamDecoder(ser)
		en := msg.CborTranscoder{}
		m, ok := sd.DecodeNext()
		assert.True(t, ok)
		assert.Equal(t, msg.MyVersion, m.Version)
		assert.Nil(t, m.IdReq)
		assert.Nil(t, m.IdRes)
		assert.Nil(t, m.ListReq)
		assert.Nil(t, m.ListRes)
		assert.NotNil(t, m.RelayReq)
		assert.Nil(t, m.RelayRes)
		assert.Nil(t, m.RelayInd)
		assert.Equal(t, []byte{0x00, 0x11, 0x22, 0x33}, m.RelayReq.Msg)
		assert.Equal(t, []msg.ClientId{1, 2, 3, 4, 5}, m.RelayReq.Dest)
		rsp := msg.Message{
			Version:   msg.MyVersion,
			MessageId: m.MessageId,
			RelayRes:  &msg.RelayResponse{Status: msg.SUCCESS, StatusMap: msg.ClientStatusMap{2: msg.INVALID_ID, 3: msg.CONNECTION_ERROR}},
		}
		rspb, ok := en.Encode(rsp)
		assert.True(t, ok)
		n, err := ser.Write(rspb)
		assert.Equal(t, len(rspb), n)
		assert.Nil(t, err)
	}()

	tc := NewClient(cli)
	csm, status := tc.RelayMessage([]byte{0x00, 0x11, 0x22, 0x33}, []msg.ClientId{1, 2, 3, 4, 5})
	assert.Equal(t, msg.SUCCESS, status)
	assert.Equal(t, msg.ClientStatusMap{2: msg.INVALID_ID, 3: msg.CONNECTION_ERROR}, csm)
}

func TestClientRelayInd(t *testing.T) {
	cli, ser := net.Pipe()

	// Fake server to send the relay indication
	go func() {
		en := msg.CborTranscoder{}
		ind := msg.Message{
			Version:   msg.MyVersion,
			MessageId: 1,
			RelayInd: &msg.RelayIndication{
				Src: msg.ClientId(888),
				Msg: []byte{11, 22, 33},
			},
		}
		indb, ok := en.Encode(ind)
		assert.True(t, ok)
		n, err := ser.Write(indb)
		assert.Equal(t, len(indb), n)
		assert.Nil(t, err)
	}()

	tc := NewClient(cli)
	mesg, ok := <-tc.Relays
	assert.True(t, ok)
	assert.Equal(t, msg.ClientId(888), mesg.Src)
	assert.Equal(t, []byte{11, 22, 33}, mesg.Msg)
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

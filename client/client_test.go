package client

import (
	"testing"
)

func TestClientBasic(t *testing.T) {
	tc = client.NewClient()
	tc.GetClientId()
}

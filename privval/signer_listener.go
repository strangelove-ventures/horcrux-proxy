package privval

import (
	"net"

	"github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
)

type SignerListener struct {
	address string
	*SignerListenerEndpoint
}

func NewSignerListener(logger cometlog.Logger, address string) SignerListener {
	proto, address := cometnet.ProtocolAndAddress(address)

	ln, err := net.Listen(proto, address)
	logger.Info("SignerListener: Listening", "proto", proto, "address", address)
	if err != nil {
		panic(err)
	}

	var listener net.Listener

	if proto == "unix" {
		unixLn := NewUnixListener(ln)
		listener = unixLn
	} else {
		tcpLn := NewTCPListener(ln, ed25519.GenPrivKey())
		listener = tcpLn
	}

	return SignerListener{
		address:                address,
		SignerListenerEndpoint: NewSignerListenerEndpoint(logger, listener),
	}
}

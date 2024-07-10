package signer

import (
	"io"
	"net"
	"time"

	cometcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
	"github.com/cometbft/cometbft/libs/protoio"
	cometservice "github.com/cometbft/cometbft/libs/service"
	cometp2pconn "github.com/cometbft/cometbft/p2p/conn"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
)

const sleep = 1

type HorcruxConnection interface {
	SendRequest(request cometprotoprivval.Message) (*cometprotoprivval.Message, error)
}

// ReconnRemoteSigner dials using its dialer and responds to any
// signature requests using its privVal.
type ReconnRemoteSigner struct {
	cometservice.BaseService

	address string
	privKey cometcryptoed25519.PrivKey

	horcruxConnection HorcruxConnection

	dialer net.Dialer

	maxReadSize int
}

// NewReconnRemoteSigner return a ReconnRemoteSigner that will dial using the given
// dialer and respond to any signature requests over the connection
// using the given privVal.
//
// If the connection is broken, the ReconnRemoteSigner will attempt to reconnect.
func NewReconnRemoteSigner(
	address string,
	logger cometlog.Logger,
	horcruxConnection HorcruxConnection,
	dialer net.Dialer,
	maxReadSize int,
) *ReconnRemoteSigner {
	rs := &ReconnRemoteSigner{
		address:           address,
		dialer:            dialer,
		horcruxConnection: horcruxConnection,
		privKey:           cometcryptoed25519.GenPrivKey(),
		maxReadSize:       maxReadSize,
	}

	rs.BaseService = *cometservice.NewBaseService(logger, "RemoteSigner", rs)
	return rs
}

// OnStart implements cmn.Service.
func (rs *ReconnRemoteSigner) OnStart() error {
	go rs.loop()
	return nil
}

// OnStop implements cmn.Service.
func (rs *ReconnRemoteSigner) OnStop() {
}

// main loop for ReconnRemoteSigner
func (rs *ReconnRemoteSigner) loop() {
	var conn net.Conn
	for {
		if !rs.IsRunning() {
			if conn != nil {
				if err := conn.Close(); err != nil {
					rs.Logger.Error("Close", "err", err.Error()+"closing listener failed")
				}
			}
			return
		}

		for conn == nil {
			if !rs.IsRunning() {
				return
			}
			proto, address := cometnet.ProtocolAndAddress(rs.address)
			netConn, err := rs.dialer.Dial(proto, address)
			if err != nil {
				rs.Logger.Error("Dialing", "err", err)
				rs.Logger.Info("Retrying", "sleep (s)", sleep, "address", rs.address)
				time.Sleep(time.Second * time.Duration(sleep))
				continue
			}

			rs.Logger.Info("Connected to Sentry", "address", rs.address)
			conn, err = cometp2pconn.MakeSecretConnection(netConn, rs.privKey)
			if err != nil {
				if err := netConn.Close(); err != nil {
					rs.Logger.Error("Error closing netConn", "err", err)
				}
				conn = nil
				rs.Logger.Error("Secret Conn", "err", err)
				rs.Logger.Info("Retrying", "sleep (s)", sleep, "address", rs.address)
				time.Sleep(time.Second * time.Duration(sleep))
				continue
			}
		}

		// since dialing can take time, we check running again
		if !rs.IsRunning() {
			if err := conn.Close(); err != nil {
				rs.Logger.Error("Close", "err", err.Error()+"closing listener failed")
			}
			return
		}

		req, err := ReadMsg(conn, rs.maxReadSize)
		if err != nil {
			rs.Logger.Error("readMsg", "err", err)
			conn.Close()
			conn = nil
			continue
		}

		// handleRequest handles request errors. We always send back a response
		res, err := rs.horcruxConnection.SendRequest(req)
		if err != nil {
			rs.Logger.Error("handleRequest", "err", err)
			conn.Close()
			conn = nil
			continue
		}

		if res == nil {
			rs.Logger.Error("handleRequest", "err", "nil response")
			conn.Close()
			conn = nil
			continue
		}

		err = WriteMsg(conn, *res)
		if err != nil {
			rs.Logger.Error("writeMsg", "err", err)
			conn.Close()
			conn = nil
		}
	}
}

// ReadMsg reads a message from an io.Reader
func ReadMsg(reader io.Reader, maxReadSize int) (msg cometprotoprivval.Message, err error) {
	if maxReadSize <= 0 {
		maxReadSize = 1024 * 1024 // 1MB
	}
	protoReader := protoio.NewDelimitedReader(reader, maxReadSize)
	_, err = protoReader.ReadMsg(&msg)
	return msg, err
}

// WriteMsg writes a message to an io.Writer
func WriteMsg(writer io.Writer, msg cometprotoprivval.Message) (err error) {
	protoWriter := protoio.NewDelimitedWriter(writer)
	_, err = protoWriter.WriteMsg(&msg)
	return err
}

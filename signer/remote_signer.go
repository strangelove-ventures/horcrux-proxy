package signer

import (
	"fmt"
	"io"
	"net"
	"time"

	cometcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	cometnet "github.com/cometbft/cometbft/libs/net"
	"github.com/cometbft/cometbft/libs/protoio"
	cometservice "github.com/cometbft/cometbft/libs/service"
	cometp2pconn "github.com/cometbft/cometbft/p2p/conn"
	cometprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	cometproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/strangelove-ventures/horcrux-proxy/privval"
)

// ReconnRemoteSigner dials using its dialer and responds to any
// signature requests using its privVal.
type ReconnRemoteSigner struct {
	cometservice.BaseService

	address string
	privKey cometcryptoed25519.PrivKey

	loadBalancer *privval.RemoteSignerLoadBalancer

	dialer net.Dialer
}

// NewReconnRemoteSigner return a ReconnRemoteSigner that will dial using the given
// dialer and respond to any signature requests over the connection
// using the given privVal.
//
// If the connection is broken, the ReconnRemoteSigner will attempt to reconnect.
func NewReconnRemoteSigner(
	address string,
	logger cometlog.Logger,
	loadBalancer *privval.RemoteSignerLoadBalancer,
	dialer net.Dialer,
) *ReconnRemoteSigner {
	rs := &ReconnRemoteSigner{
		address:      address,
		dialer:       dialer,
		loadBalancer: loadBalancer,
		privKey:      cometcryptoed25519.GenPrivKey(),
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
				rs.Logger.Info("Retrying", "sleep (s)", 3, "address", rs.address)
				time.Sleep(time.Second * 3)
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
				rs.Logger.Info("Retrying", "sleep (s)", 3, "address", rs.address)
				time.Sleep(time.Second * 3)
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

		req, err := ReadMsg(conn)
		if err != nil {
			rs.Logger.Error("readMsg", "err", err)
			conn.Close()
			conn = nil
			continue
		}

		// handleRequest handles request errors. We always send back a response
		res := rs.handleRequest(req)

		err = WriteMsg(conn, res)
		if err != nil {
			rs.Logger.Error("writeMsg", "err", err)
			conn.Close()
			conn = nil
		}
	}
}

func (rs *ReconnRemoteSigner) handleRequest(req cometprotoprivval.Message) cometprotoprivval.Message {
	switch typedReq := req.Sum.(type) {
	case *cometprotoprivval.Message_SignVoteRequest:
		return rs.handleSignVoteRequest(req)
	case *cometprotoprivval.Message_SignProposalRequest:
		return rs.handleSignProposalRequest(req)
	case *cometprotoprivval.Message_PubKeyRequest:
		return rs.handlePubKeyRequest(req)
	case *cometprotoprivval.Message_PingRequest:
		return rs.handlePingRequest()
	default:
		rs.Logger.Error("Unknown request", "err", fmt.Errorf("%v", typedReq))
		return cometprotoprivval.Message{}
	}
}

func (rs *ReconnRemoteSigner) handleSignVoteRequest(req cometprotoprivval.Message) cometprotoprivval.Message {
	res, err := rs.loadBalancer.SendRequest(req)
	if err == nil {
		return *res
	}

	return cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_SignedVoteResponse{SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
			Vote:  cometproto.Vote{},
			Error: getRemoteSignerError(err),
		}},
	}
}

func (rs *ReconnRemoteSigner) handleSignProposalRequest(req cometprotoprivval.Message) cometprotoprivval.Message {
	res, err := rs.loadBalancer.SendRequest(req)
	if err == nil {
		return *res
	}

	return cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_SignedProposalResponse{
			SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
				Proposal: cometproto.Proposal{},
				Error:    getRemoteSignerError(err),
			}},
	}
}

func (rs *ReconnRemoteSigner) handlePubKeyRequest(req cometprotoprivval.Message) cometprotoprivval.Message {
	res, err := rs.loadBalancer.SendRequest(req)
	if err == nil {
		return *res
	}

	return cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PubKeyResponse{PubKeyResponse: &cometprotoprivval.PubKeyResponse{
			PubKey: cometprotocrypto.PublicKey{},
			Error:  getRemoteSignerError(err),
		}},
	}
}

func (rs *ReconnRemoteSigner) handlePingRequest() cometprotoprivval.Message {
	return cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PingResponse{
			PingResponse: &cometprotoprivval.PingResponse{},
		},
	}
}

func getRemoteSignerError(err error) *cometprotoprivval.RemoteSignerError {
	if err == nil {
		return nil
	}
	return &cometprotoprivval.RemoteSignerError{
		Code:        0,
		Description: err.Error(),
	}
}

// ReadMsg reads a message from an io.Reader
func ReadMsg(reader io.Reader) (msg cometprotoprivval.Message, err error) {
	const maxRemoteSignerMsgSize = 1024 * 10
	protoReader := protoio.NewDelimitedReader(reader, maxRemoteSignerMsgSize)
	_, err = protoReader.ReadMsg(&msg)
	return msg, err
}

// WriteMsg writes a message to an io.Writer
func WriteMsg(writer io.Writer, msg cometprotoprivval.Message) (err error) {
	protoWriter := protoio.NewDelimitedWriter(writer)
	_, err = protoWriter.WriteMsg(&msg)
	return err
}

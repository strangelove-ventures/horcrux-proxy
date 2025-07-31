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
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	"github.com/strangelove-ventures/horcrux-proxy/privval"
)

// ReconnRemoteSignerV2 dials using its dialer and responds to any
// signature requests using its privVal with protocol version support.
type ReconnRemoteSignerV2 struct {
	cometservice.BaseService

	address         string
	privKey         cometcryptoed25519.PrivKey
	protocol        privval.PrivvalProtocol
	protocolVersion string

	horcruxConnection HorcruxConnection

	dialer net.Dialer

	maxReadSize int
}

// NewReconnRemoteSignerV2 return a ReconnRemoteSignerV2 that will dial using the given
// dialer and respond to any signature requests over the connection
// using the given privVal with the specified protocol version.
func NewReconnRemoteSignerV2(
	address string,
	logger cometlog.Logger,
	horcruxConnection HorcruxConnection,
	dialer net.Dialer,
	maxReadSize int,
	protocolVersion string,
) (*ReconnRemoteSignerV2, error) {
	protocol, err := privval.GetProtocol(protocolVersion)
	if err != nil {
		return nil, err
	}

	rs := &ReconnRemoteSignerV2{
		address:           address,
		dialer:            dialer,
		horcruxConnection: horcruxConnection,
		privKey:           cometcryptoed25519.GenPrivKey(),
		maxReadSize:       maxReadSize,
		protocol:          protocol,
		protocolVersion:   protocolVersion,
	}

	rs.BaseService = *cometservice.NewBaseService(logger, "RemoteSignerV2", rs)
	return rs, nil
}

// OnStart implements cmn.Service.
func (rs *ReconnRemoteSignerV2) OnStart() error {
	go rs.loop()
	return nil
}

// OnStop implements cmn.Service.
func (rs *ReconnRemoteSignerV2) OnStop() {
}

// main loop for ReconnRemoteSignerV2
func (rs *ReconnRemoteSignerV2) loop() {
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

			rs.Logger.Info("Connected to Sentry", "address", rs.address, "protocol", rs.protocolVersion)
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

		var err error
		if rs.protocolVersion == privval.ProtocolVersionV1 {
			err = rs.HandleV1Connection(conn)
		} else {
			err = rs.HandleLegacyConnection(conn)
		}

		if err != nil {
			rs.Logger.Error("handleConnection", "err", err)
			conn.Close()
			conn = nil
		}
	}
}

func (rs *ReconnRemoteSignerV2) HandleLegacyConnection(conn net.Conn) error {
	req, err := ReadMsg(conn, rs.maxReadSize)
	if err != nil {
		return fmt.Errorf("readMsg: %w", err)
	}

	// handleRequest handles request errors. We always send back a response
	res, err := rs.horcruxConnection.SendRequest(req)
	if err != nil {
		return fmt.Errorf("handleRequest: %w", err)
	}

	if res == nil {
		return fmt.Errorf("handleRequest: nil response")
	}

	err = WriteMsg(conn, *res)
	if err != nil {
		return fmt.Errorf("writeMsg: %w", err)
	}

	return nil
}

func (rs *ReconnRemoteSignerV2) HandleV1Connection(conn net.Conn) error {
	// Read v1 message
	msg, err := rs.readV1Msg(conn)
	if err != nil {
		return fmt.Errorf("readV1Msg: %w", err)
	}

	// Convert v1 message to legacy format and process
	legacyReq, err := rs.ConvertV1ToLegacyRequest(msg)
	if err != nil {
		return fmt.Errorf("ConvertV1ToLegacyRequest: %w", err)
	}

	// Send to horcrux
	legacyRes, err := rs.horcruxConnection.SendRequest(*legacyReq)
	if err != nil {
		return fmt.Errorf("handleRequest: %w", err)
	}

	if legacyRes == nil {
		return fmt.Errorf("handleRequest: nil response")
	}

	// Convert legacy response back to v1 format
	v1Res, err := rs.ConvertLegacyToV1Response(legacyRes)
	if err != nil {
		return fmt.Errorf("ConvertLegacyToV1Response: %w", err)
	}

	// Write v1 response
	err = rs.writeV1Msg(conn, v1Res)
	if err != nil {
		return fmt.Errorf("writeV1Msg: %w", err)
	}

	return nil
}

func (rs *ReconnRemoteSignerV2) readV1Msg(reader io.Reader) (*privval.V1Message, error) {
	if rs.maxReadSize <= 0 {
		rs.maxReadSize = 1024 * 1024 // 1MB
	}
	protoReader := protoio.NewDelimitedReader(reader, rs.maxReadSize)
	var msg privval.V1Message
	_, err := protoReader.ReadMsg(&msg)
	return &msg, err
}

func (rs *ReconnRemoteSignerV2) writeV1Msg(writer io.Writer, msg *privval.V1Message) error {
	protoWriter := protoio.NewDelimitedWriter(writer)
	_, err := protoWriter.WriteMsg(msg)
	return err
}

func (rs *ReconnRemoteSignerV2) ConvertV1ToLegacyRequest(v1Msg *privval.V1Message) (*cometprotoprivval.Message, error) {
	var legacyMsg cometprotoprivval.Message

	switch msg := v1Msg.Sum.(type) {
	case *privval.V1Message_PubKeyRequest:
		req, err := rs.protocol.ConvertPubKeyRequest(msg.PubKeyRequest)
		if err != nil {
			return nil, err
		}
		legacyMsg.Sum = &cometprotoprivval.Message_PubKeyRequest{
			PubKeyRequest: req,
		}

	case *privval.V1Message_SignVoteRequest:
		req, err := rs.protocol.ConvertSignVoteRequest(msg.SignVoteRequest)
		if err != nil {
			return nil, err
		}
		legacyMsg.Sum = &cometprotoprivval.Message_SignVoteRequest{
			SignVoteRequest: req,
		}

	case *privval.V1Message_SignProposalRequest:
		req, err := rs.protocol.ConvertSignProposalRequest(msg.SignProposalRequest)
		if err != nil {
			return nil, err
		}
		legacyMsg.Sum = &cometprotoprivval.Message_SignProposalRequest{
			SignProposalRequest: req,
		}

	case *privval.V1Message_PingRequest:
		req, err := rs.protocol.ConvertPingRequest(msg.PingRequest)
		if err != nil {
			return nil, err
		}
		legacyMsg.Sum = &cometprotoprivval.Message_PingRequest{
			PingRequest: req,
		}

	default:
		return nil, fmt.Errorf("unknown v1 message type: %T", msg)
	}

	return &legacyMsg, nil
}

func (rs *ReconnRemoteSignerV2) ConvertLegacyToV1Response(legacyMsg *cometprotoprivval.Message) (*privval.V1Message, error) {
	var v1Msg privval.V1Message

	switch msg := legacyMsg.Sum.(type) {
	case *cometprotoprivval.Message_PubKeyResponse:
		res, err := rs.protocol.ConvertPubKeyResponse(&msg.PubKeyResponse.PubKey, convertRemoteSignerError(msg.PubKeyResponse.Error))
		if err != nil {
			return nil, err
		}
		if v1Res, ok := res.(*privval.V1PubKeyResponse); ok {
			v1Msg.Sum = &privval.V1Message_PubKeyResponse{
				PubKeyResponse: v1Res,
			}
		} else {
			return nil, fmt.Errorf("invalid PubKeyResponse conversion")
		}

	case *cometprotoprivval.Message_SignedVoteResponse:
		res, err := rs.protocol.ConvertSignVoteResponse(&msg.SignedVoteResponse.Vote, convertRemoteSignerError(msg.SignedVoteResponse.Error))
		if err != nil {
			return nil, err
		}
		if v1Res, ok := res.(*privval.V1SignedVoteResponse); ok {
			v1Msg.Sum = &privval.V1Message_SignedVoteResponse{
				SignedVoteResponse: v1Res,
			}
		} else {
			return nil, fmt.Errorf("invalid SignedVoteResponse conversion")
		}

	case *cometprotoprivval.Message_SignedProposalResponse:
		res, err := rs.protocol.ConvertSignProposalResponse(&msg.SignedProposalResponse.Proposal, convertRemoteSignerError(msg.SignedProposalResponse.Error))
		if err != nil {
			return nil, err
		}
		if v1Res, ok := res.(*privval.V1SignedProposalResponse); ok {
			v1Msg.Sum = &privval.V1Message_SignedProposalResponse{
				SignedProposalResponse: v1Res,
			}
		} else {
			return nil, fmt.Errorf("invalid SignedProposalResponse conversion")
		}

	case *cometprotoprivval.Message_PingResponse:
		res, err := rs.protocol.ConvertPingResponse()
		if err != nil {
			return nil, err
		}
		if v1Res, ok := res.(*privval.V1PingResponse); ok {
			v1Msg.Sum = &privval.V1Message_PingResponse{
				PingResponse: v1Res,
			}
		} else {
			return nil, fmt.Errorf("invalid PingResponse conversion")
		}

	default:
		return nil, fmt.Errorf("unknown legacy message type: %T", msg)
	}

	return &v1Msg, nil
}

func convertRemoteSignerError(err *cometprotoprivval.RemoteSignerError) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s", err.Description)
}
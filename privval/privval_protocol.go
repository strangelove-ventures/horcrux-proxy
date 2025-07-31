package privval

import (
	"fmt"

	cometprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	cometproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// Protocol version constants
const (
	ProtocolVersionLegacy = "legacy"
	ProtocolVersionV1     = "v1"
)

// PrivvalProtocol interface for handling different protocol versions
type PrivvalProtocol interface {
	// Convert messages between protocol versions
	ConvertPubKeyRequest(req interface{}) (*cometprotoprivval.PubKeyRequest, error)
	ConvertPubKeyResponse(pubKey *cometprotocrypto.PublicKey, err error) (interface{}, error)
	ConvertSignVoteRequest(req interface{}) (*cometprotoprivval.SignVoteRequest, error)
	ConvertSignVoteResponse(vote *cometproto.Vote, err error) (interface{}, error)
	ConvertSignProposalRequest(req interface{}) (*cometprotoprivval.SignProposalRequest, error)
	ConvertSignProposalResponse(proposal *cometproto.Proposal, err error) (interface{}, error)
	ConvertPingRequest(req interface{}) (*cometprotoprivval.PingRequest, error)
	ConvertPingResponse() (interface{}, error)

	// Handle protocol-specific message types
	HandleMessage(msg interface{}) (interface{}, error)
	GetProtocolVersion() string
}

// LegacyProtocol implements the legacy (v0.38) protocol
type LegacyProtocol struct{}

func NewLegacyProtocol() PrivvalProtocol {
	return &LegacyProtocol{}
}

func (p *LegacyProtocol) GetProtocolVersion() string {
	return ProtocolVersionLegacy
}

func (p *LegacyProtocol) ConvertPubKeyRequest(req interface{}) (*cometprotoprivval.PubKeyRequest, error) {
	if msg, ok := req.(*cometprotoprivval.PubKeyRequest); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid PubKeyRequest type for legacy protocol")
}

func (p *LegacyProtocol) ConvertPubKeyResponse(pubKey *cometprotocrypto.PublicKey, err error) (interface{}, error) {
	var remoteError *cometprotoprivval.RemoteSignerError
	if err != nil {
		remoteError = &cometprotoprivval.RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}
	
	var pk cometprotocrypto.PublicKey
	if pubKey != nil {
		pk = *pubKey
	}
	
	return &cometprotoprivval.PubKeyResponse{
		PubKey: pk,
		Error:  remoteError,
	}, nil
}

func (p *LegacyProtocol) ConvertSignVoteRequest(req interface{}) (*cometprotoprivval.SignVoteRequest, error) {
	if msg, ok := req.(*cometprotoprivval.SignVoteRequest); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid SignVoteRequest type for legacy protocol")
}

func (p *LegacyProtocol) ConvertSignVoteResponse(vote *cometproto.Vote, err error) (interface{}, error) {
	var remoteError *cometprotoprivval.RemoteSignerError
	if err != nil {
		remoteError = &cometprotoprivval.RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}
	
	resp := &cometprotoprivval.SignedVoteResponse{
		Error: remoteError,
	}
	
	if vote != nil {
		resp.Vote = *vote
	}
	
	return resp, nil
}

func (p *LegacyProtocol) ConvertSignProposalRequest(req interface{}) (*cometprotoprivval.SignProposalRequest, error) {
	if msg, ok := req.(*cometprotoprivval.SignProposalRequest); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid SignProposalRequest type for legacy protocol")
}

func (p *LegacyProtocol) ConvertSignProposalResponse(proposal *cometproto.Proposal, err error) (interface{}, error) {
	var remoteError *cometprotoprivval.RemoteSignerError
	if err != nil {
		remoteError = &cometprotoprivval.RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}
	
	resp := &cometprotoprivval.SignedProposalResponse{
		Error: remoteError,
	}
	
	if proposal != nil {
		resp.Proposal = *proposal
	}
	
	return resp, nil
}

func (p *LegacyProtocol) ConvertPingRequest(req interface{}) (*cometprotoprivval.PingRequest, error) {
	if msg, ok := req.(*cometprotoprivval.PingRequest); ok {
		return msg, nil
	}
	return nil, fmt.Errorf("invalid PingRequest type for legacy protocol")
}

func (p *LegacyProtocol) ConvertPingResponse() (interface{}, error) {
	return &cometprotoprivval.PingResponse{}, nil
}

func (p *LegacyProtocol) HandleMessage(msg interface{}) (interface{}, error) {
	// Legacy protocol uses the standard cometprotoprivval.Message
	if privvalMsg, ok := msg.(*cometprotoprivval.Message); ok {
		return privvalMsg, nil
	}
	return nil, fmt.Errorf("invalid message type for legacy protocol")
}

// GetProtocol returns the appropriate protocol handler based on version
func GetProtocol(version string) (PrivvalProtocol, error) {
	switch version {
	case ProtocolVersionLegacy, "":
		return NewLegacyProtocol(), nil
	case ProtocolVersionV1:
		return NewV1Protocol(), nil
	default:
		return nil, fmt.Errorf("unknown protocol version: %s", version)
	}
}
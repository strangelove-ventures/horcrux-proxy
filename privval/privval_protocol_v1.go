package privval

import (
	"fmt"

	cometprotocrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	cometproto "github.com/cometbft/cometbft/proto/tendermint/types"
)

// V1Protocol implements the CometBFT v1.0 protocol
type V1Protocol struct{}

func NewV1Protocol() PrivvalProtocol {
	return &V1Protocol{}
}

func (p *V1Protocol) GetProtocolVersion() string {
	return ProtocolVersionV1
}

func (p *V1Protocol) ConvertPubKeyRequest(req interface{}) (*cometprotoprivval.PubKeyRequest, error) {
	if v1Req, ok := req.(*V1PubKeyRequest); ok {
		// v1 PubKeyRequest has the same structure, just convert
		return &cometprotoprivval.PubKeyRequest{
			ChainId: v1Req.ChainId,
		}, nil
	}
	return nil, fmt.Errorf("invalid PubKeyRequest type for v1 protocol")
}

func (p *V1Protocol) ConvertPubKeyResponse(pubKey *cometprotocrypto.PublicKey, err error) (interface{}, error) {
	var v1Error *V1RemoteSignerError
	if err != nil {
		v1Error = &V1RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}

	// Convert legacy public key to v1 format
	var pubKeyBytes []byte
	var pubKeyType string

	if pubKey != nil && pubKey.Sum != nil {
		switch pk := pubKey.Sum.(type) {
		case *cometprotocrypto.PublicKey_Ed25519:
			pubKeyBytes = pk.Ed25519
			pubKeyType = "ed25519"
		default:
			return nil, fmt.Errorf("unsupported public key type")
		}
	}

	return &V1PubKeyResponse{
		PubKeyBytes: pubKeyBytes,
		PubKeyType:  pubKeyType,
		Error:       v1Error,
	}, nil
}

func (p *V1Protocol) ConvertSignVoteRequest(req interface{}) (*cometprotoprivval.SignVoteRequest, error) {
	if v1Req, ok := req.(*V1SignVoteRequest); ok {
		// Convert v1 Vote to legacy format
		vote := &cometproto.Vote{
			Type:   cometproto.SignedMsgType(v1Req.Vote.Type),
			Height: v1Req.Vote.Height,
			Round:  v1Req.Vote.Round,
			BlockID: cometproto.BlockID{
				Hash: v1Req.Vote.BlockId.Hash,
				PartSetHeader: cometproto.PartSetHeader{
					Total: v1Req.Vote.BlockId.PartSetHeader.Total,
					Hash:  v1Req.Vote.BlockId.PartSetHeader.Hash,
				},
			},
			Timestamp:          *v1Req.Vote.Timestamp,
			ValidatorAddress:   v1Req.Vote.ValidatorAddress,
			ValidatorIndex:     v1Req.Vote.ValidatorIndex,
			Signature:          v1Req.Vote.Signature,
			Extension:          v1Req.Vote.Extension,
			ExtensionSignature: v1Req.Vote.ExtensionSignature,
		}

		return &cometprotoprivval.SignVoteRequest{
			Vote:    vote,
			ChainId: v1Req.ChainId,
		}, nil
	}
	return nil, fmt.Errorf("invalid SignVoteRequest type for v1 protocol")
}

func (p *V1Protocol) ConvertSignVoteResponse(vote *cometproto.Vote, err error) (interface{}, error) {
	var v1Error *V1RemoteSignerError
	if err != nil {
		v1Error = &V1RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}

	// Convert legacy vote to v1 format
	var v1Vote *V1Vote
	if vote != nil {
		v1Vote = &V1Vote{
			Type:   V1SignedMsgType(vote.Type),
			Height: vote.Height,
			Round:  vote.Round,
			BlockId: &V1BlockID{
				Hash: vote.BlockID.Hash,
				PartSetHeader: &V1PartSetHeader{
					Total: vote.BlockID.PartSetHeader.Total,
					Hash:  vote.BlockID.PartSetHeader.Hash,
				},
			},
			Timestamp:          &vote.Timestamp,
			ValidatorAddress:   vote.ValidatorAddress,
			ValidatorIndex:     vote.ValidatorIndex,
			Signature:          vote.Signature,
			Extension:          vote.Extension,
			ExtensionSignature: vote.ExtensionSignature,
		}
	}

	return &V1SignedVoteResponse{
		Vote:  v1Vote,
		Error: v1Error,
	}, nil
}

func (p *V1Protocol) ConvertSignProposalRequest(req interface{}) (*cometprotoprivval.SignProposalRequest, error) {
	if v1Req, ok := req.(*V1SignProposalRequest); ok {
		// Convert v1 Proposal to legacy format
		proposal := &cometproto.Proposal{
			Type:     cometproto.SignedMsgType(v1Req.Proposal.Type),
			Height:   v1Req.Proposal.Height,
			Round:    v1Req.Proposal.Round,
			PolRound: v1Req.Proposal.PolRound,
			BlockID: cometproto.BlockID{
				Hash: v1Req.Proposal.BlockId.Hash,
				PartSetHeader: cometproto.PartSetHeader{
					Total: v1Req.Proposal.BlockId.PartSetHeader.Total,
					Hash:  v1Req.Proposal.BlockId.PartSetHeader.Hash,
				},
			},
			Timestamp: *v1Req.Proposal.Timestamp,
			Signature: v1Req.Proposal.Signature,
		}

		return &cometprotoprivval.SignProposalRequest{
			Proposal: proposal,
			ChainId:  v1Req.ChainId,
		}, nil
	}
	return nil, fmt.Errorf("invalid SignProposalRequest type for v1 protocol")
}

func (p *V1Protocol) ConvertSignProposalResponse(proposal *cometproto.Proposal, err error) (interface{}, error) {
	var v1Error *V1RemoteSignerError
	if err != nil {
		v1Error = &V1RemoteSignerError{
			Code:        0,
			Description: err.Error(),
		}
	}

	// Convert legacy proposal to v1 format
	var v1Proposal *V1Proposal
	if proposal != nil {
		v1Proposal = &V1Proposal{
			Type:     V1SignedMsgType(proposal.Type),
			Height:   proposal.Height,
			Round:    proposal.Round,
			PolRound: proposal.PolRound,
			BlockId: &V1BlockID{
				Hash: proposal.BlockID.Hash,
				PartSetHeader: &V1PartSetHeader{
					Total: proposal.BlockID.PartSetHeader.Total,
					Hash:  proposal.BlockID.PartSetHeader.Hash,
				},
			},
			Timestamp: &proposal.Timestamp,
			Signature: proposal.Signature,
		}
	}

	return &V1SignedProposalResponse{
		Proposal: v1Proposal,
		Error:    v1Error,
	}, nil
}

func (p *V1Protocol) ConvertPingRequest(req interface{}) (*cometprotoprivval.PingRequest, error) {
	if _, ok := req.(*V1PingRequest); ok {
		// v1 PingRequest has the same structure
		return &cometprotoprivval.PingRequest{}, nil
	}
	return nil, fmt.Errorf("invalid PingRequest type for v1 protocol")
}

func (p *V1Protocol) ConvertPingResponse() (interface{}, error) {
	return &V1PingResponse{}, nil
}

func (p *V1Protocol) HandleMessage(msg interface{}) (interface{}, error) {
	// V1 protocol uses V1Message
	if v1Msg, ok := msg.(*V1Message); ok {
		return v1Msg, nil
	}
	return nil, fmt.Errorf("invalid message type for v1 protocol")
}

// Helper function to convert v1 public key response to legacy format
func ConvertV1PubKeyToLegacy(pubKeyBytes []byte, pubKeyType string) (*cometprotocrypto.PublicKey, error) {
	switch pubKeyType {
	case "ed25519":
		return &cometprotocrypto.PublicKey{
			Sum: &cometprotocrypto.PublicKey_Ed25519{
				Ed25519: pubKeyBytes,
			},
		}, nil
	default:
		return nil, fmt.Errorf("unsupported public key type: %s", pubKeyType)
	}
}
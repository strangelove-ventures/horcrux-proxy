package signer

import (
	"context"
	"fmt"
	"time"

	cometlog "github.com/cometbft/cometbft/libs/log"
	cometcrypto "github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	"github.com/strangelove-ventures/horcrux/v3/signer"
	"github.com/strangelove-ventures/horcrux/v3/signer/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ HorcruxConnection = (*HorcruxGRPCClient)(nil)

type HorcruxGRPCClient struct {
	grpcClient proto.RemoteSignerClient
	logger     cometlog.Logger
}

func NewHorcruxGRPCClient(
	logger cometlog.Logger,
	address string,
) (*HorcruxGRPCClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &HorcruxGRPCClient{
		logger:     logger,
		grpcClient: proto.NewRemoteSignerClient(conn),
	}, nil
}

func (c *HorcruxGRPCClient) SendRequest(req cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	switch typedReq := req.Sum.(type) {
	case *cometprotoprivval.Message_SignVoteRequest:
		return c.handleSignVoteRequest(req)
	case *cometprotoprivval.Message_SignProposalRequest:
		return c.handleSignProposalRequest(req)
	case *cometprotoprivval.Message_PubKeyRequest:
		return c.handlePubKeyRequest(req)
	case *cometprotoprivval.Message_PingRequest:
		return c.handlePingRequest()
	default:
		c.logger.Error("Unknown request", "err", fmt.Errorf("%v", typedReq))
		return &cometprotoprivval.Message{}, nil
	}
}

func (c *HorcruxGRPCClient) handleSignVoteRequest(req cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	voteReq := req.GetSignVoteRequest()
	vote := voteReq.Vote

	res, err := c.grpcClient.Sign(context.TODO(), &proto.SignBlockRequest{
		ChainID: voteReq.ChainId,
		Block:   signer.VoteToBlock(voteReq.ChainId, vote).ToProto(),
	})
	if err == nil {
		vote.Signature = res.Signature
		vote.ExtensionSignature = res.VoteExtSignature
		vote.Timestamp = time.Unix(0, res.Timestamp)
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedVoteResponse{
				SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
					Vote: *vote,
				},
			},
		}, nil
	}

	return &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_SignedVoteResponse{SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
			Error: getRemoteSignerError(err),
		}},
	}, nil
}

func (c *HorcruxGRPCClient) handleSignProposalRequest(req cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	proposalReq := req.GetSignProposalRequest()
	proposal := proposalReq.Proposal
	res, err := c.grpcClient.Sign(context.TODO(), &proto.SignBlockRequest{
		ChainID: proposalReq.ChainId,
		Block:   signer.ProposalToBlock(proposalReq.ChainId, proposal).ToProto(),
	})
	if err == nil {
		proposal.Signature = res.Signature
		proposal.Timestamp = time.Unix(0, res.Timestamp)
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedProposalResponse{
				SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
					Proposal: *proposal,
				},
			},
		}, nil
	}

	return &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_SignedProposalResponse{SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
			Error: getRemoteSignerError(err),
		}},
	}, nil
}

func (c *HorcruxGRPCClient) handlePubKeyRequest(req cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	res, err := c.grpcClient.PubKey(context.TODO(), &proto.PubKeyRequest{
		ChainId: req.GetPubKeyRequest().ChainId,
	})
	if err == nil {
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					PubKey: cometcrypto.PublicKey{
						Sum: &cometcrypto.PublicKey_Ed25519{
							Ed25519: res.PubKey,
						},
					},
				},
			},
		}, nil
	}

	return &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PubKeyResponse{PubKeyResponse: &cometprotoprivval.PubKeyResponse{
			Error: getRemoteSignerError(err),
		}},
	}, nil
}

func (c *HorcruxGRPCClient) handlePingRequest() (*cometprotoprivval.Message, error) {
	return &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PingResponse{
			PingResponse: &cometprotoprivval.PingResponse{},
		},
	}, nil
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

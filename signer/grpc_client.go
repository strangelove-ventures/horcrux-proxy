package signer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	cometprotocrypto "github.com/strangelove-ventures/horcrux/v3/comet/proto/crypto"
	cometprotoprivval "github.com/strangelove-ventures/horcrux/v3/comet/proto/privval"

	"github.com/strangelove-ventures/horcrux/v3/grpc/cosigner"
	"github.com/strangelove-ventures/horcrux/v3/grpc/horcrux"
	"github.com/strangelove-ventures/horcrux/v3/types"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var _ HorcruxConnection = (*HorcruxGRPCClient)(nil)

type HorcruxGRPCClient struct {
	grpcClient horcrux.RemoteSignerClient
	logger     *slog.Logger
}

func NewHorcruxGRPCClient(
	logger *slog.Logger,
	address string,
) (*HorcruxGRPCClient, error) {
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}
	return &HorcruxGRPCClient{
		logger:     logger,
		grpcClient: horcrux.NewRemoteSignerClient(conn),
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

	res, err := c.grpcClient.Sign(context.TODO(), &cosigner.SignBlockRequest{
		ChainID: voteReq.ChainId,
		Block:   types.VoteToBlock(vote).ToProto(),
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

	res, err := c.grpcClient.Sign(context.TODO(), &cosigner.SignBlockRequest{
		ChainID: proposalReq.ChainId,
		Block:   types.ProposalToBlock(proposal).ToProto(),
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
	res, err := c.grpcClient.PubKey(context.TODO(), &horcrux.PubKeyRequest{
		ChainId: req.GetPubKeyRequest().ChainId,
	})
	if err != nil {
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{PubKeyResponse: &cometprotoprivval.PubKeyResponse{
				Error: getRemoteSignerError(err),
			}},
		}, nil
	}

	var protoPubkey cometprotocrypto.PublicKey
	if err := protoPubkey.Unmarshal(res.PubKey); err != nil {
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{PubKeyResponse: &cometprotoprivval.PubKeyResponse{
				Error: getRemoteSignerError(err),
			}},
		}, nil
	}

	return &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PubKeyResponse{
			PubKeyResponse: &cometprotoprivval.PubKeyResponse{
				PubKey: protoPubkey,
			},
		},
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

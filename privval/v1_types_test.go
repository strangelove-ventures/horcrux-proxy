package privval_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
)

func TestV1MessageTypes(t *testing.T) {
	t.Run("V1Message_OneofTypes", func(t *testing.T) {
		// Test each oneof type
		testCases := []struct {
			name string
			msg  privval.V1Message
		}{
			{
				name: "PubKeyRequest",
				msg: privval.V1Message{
					Sum: &privval.V1Message_PubKeyRequest{
						PubKeyRequest: &privval.V1PubKeyRequest{
							ChainId: "test-chain",
						},
					},
				},
			},
			{
				name: "PubKeyResponse",
				msg: privval.V1Message{
					Sum: &privval.V1Message_PubKeyResponse{
						PubKeyResponse: &privval.V1PubKeyResponse{
							PubKeyBytes: []byte("test-key"),
							PubKeyType:  "ed25519",
						},
					},
				},
			},
			{
				name: "SignVoteRequest",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignVoteRequest{
						SignVoteRequest: &privval.V1SignVoteRequest{
							ChainId: "test-chain",
						},
					},
				},
			},
			{
				name: "SignedVoteResponse",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignedVoteResponse{
						SignedVoteResponse: &privval.V1SignedVoteResponse{},
					},
				},
			},
			{
				name: "SignProposalRequest",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignProposalRequest{
						SignProposalRequest: &privval.V1SignProposalRequest{
							ChainId: "test-chain",
						},
					},
				},
			},
			{
				name: "SignedProposalResponse",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignedProposalResponse{
						SignedProposalResponse: &privval.V1SignedProposalResponse{},
					},
				},
			},
			{
				name: "PingRequest",
				msg: privval.V1Message{
					Sum: &privval.V1Message_PingRequest{
						PingRequest: &privval.V1PingRequest{},
					},
				},
			},
			{
				name: "PingResponse",
				msg: privval.V1Message{
					Sum: &privval.V1Message_PingResponse{
						PingResponse: &privval.V1PingResponse{},
					},
				},
			},
			{
				name: "SignBytesRequest",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignBytesRequest{
						SignBytesRequest: &privval.V1SignBytesRequest{
							ChainId: "test-chain",
							Bytes:   []byte("test-bytes"),
						},
					},
				},
			},
			{
				name: "SignBytesResponse",
				msg: privval.V1Message{
					Sum: &privval.V1Message_SignBytesResponse{
						SignBytesResponse: &privval.V1SignBytesResponse{
							Signature: []byte("test-signature"),
						},
					},
				},
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test that the message is not nil
				assert.NotNil(t, tc.msg.Sum)
				
				// Test ProtoMessage interface
				tc.msg.ProtoMessage()
				tc.msg.Reset()
				str := tc.msg.String()
				assert.NotEmpty(t, str)
			})
		}
	})

	t.Run("V1PubKeyRequest", func(t *testing.T) {
		req := &privval.V1PubKeyRequest{
			ChainId: "test-chain-123",
		}

		assert.Equal(t, "test-chain-123", req.ChainId)
		
		// Test ProtoMessage interface
		req.ProtoMessage()
		req.Reset()
		assert.Empty(t, req.ChainId)
		
		req.ChainId = "test-chain-123"
		str := req.String()
		assert.NotEmpty(t, str)
	})

	t.Run("V1PubKeyResponse", func(t *testing.T) {
		resp := &privval.V1PubKeyResponse{
			PubKeyBytes: []byte("test-pubkey-bytes"),
			PubKeyType:  "ed25519",
			Error: &privval.V1RemoteSignerError{
				Code:        1,
				Description: "test error",
			},
		}

		assert.Equal(t, []byte("test-pubkey-bytes"), resp.PubKeyBytes)
		assert.Equal(t, "ed25519", resp.PubKeyType)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, int32(1), resp.Error.Code)
		assert.Equal(t, "test error", resp.Error.Description)

		// Test ProtoMessage interface
		resp.ProtoMessage()
		resp.Reset()
		assert.Nil(t, resp.PubKeyBytes)
		assert.Empty(t, resp.PubKeyType)
		assert.Nil(t, resp.Error)
	})

	t.Run("V1SignVoteRequest", func(t *testing.T) {
		timestamp := time.Now()
		req := &privval.V1SignVoteRequest{
			Vote: &privval.V1Vote{
				Type:   privval.V1SignedMsgTypePrevote,
				Height: 12345,
				Round:  2,
				BlockId: &privval.V1BlockID{
					Hash: []byte("block-hash"),
					PartSetHeader: &privval.V1PartSetHeader{
						Total: 100,
						Hash:  []byte("part-set-hash"),
					},
				},
				Timestamp:          &timestamp,
				ValidatorAddress:   []byte("validator-address"),
				ValidatorIndex:     3,
				Signature:          []byte("signature"),
				Extension:          []byte("vote-extension"),
				ExtensionSignature: []byte("ext-signature"),
			},
			ChainId:              "test-chain",
			SkipExtensionSigning: true,
		}

		assert.Equal(t, "test-chain", req.ChainId)
		assert.True(t, req.SkipExtensionSigning)
		assert.NotNil(t, req.Vote)
		assert.Equal(t, privval.V1SignedMsgTypePrevote, req.Vote.Type)
		assert.Equal(t, int64(12345), req.Vote.Height)
		assert.Equal(t, []byte("vote-extension"), req.Vote.Extension)
		assert.Equal(t, []byte("ext-signature"), req.Vote.ExtensionSignature)

		// Test ProtoMessage interface
		req.ProtoMessage()
		str := req.String()
		assert.NotEmpty(t, str)
	})

	t.Run("V1SignedVoteResponse", func(t *testing.T) {
		timestamp := time.Now()
		resp := &privval.V1SignedVoteResponse{
			Vote: &privval.V1Vote{
				Type:             privval.V1SignedMsgTypePrecommit,
				Height:           54321,
				Round:            5,
				Timestamp:        &timestamp,
				ValidatorAddress: []byte("validator"),
				ValidatorIndex:   1,
				Signature:        []byte("vote-signature"),
			},
			Error: &privval.V1RemoteSignerError{
				Code:        2,
				Description: "vote error",
			},
		}

		assert.NotNil(t, resp.Vote)
		assert.Equal(t, privval.V1SignedMsgTypePrecommit, resp.Vote.Type)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, int32(2), resp.Error.Code)

		// Test reset
		resp.Reset()
		assert.Nil(t, resp.Vote)
		assert.Nil(t, resp.Error)
	})

	t.Run("V1SignProposalRequest", func(t *testing.T) {
		timestamp := time.Now()
		req := &privval.V1SignProposalRequest{
			Proposal: &privval.V1Proposal{
				Type:     privval.V1SignedMsgTypeProposal,
				Height:   10000,
				Round:    0,
				PolRound: -1,
				BlockId: &privval.V1BlockID{
					Hash: []byte("proposal-block-hash"),
					PartSetHeader: &privval.V1PartSetHeader{
						Total: 50,
						Hash:  []byte("proposal-part-hash"),
					},
				},
				Timestamp: &timestamp,
				Signature: []byte("proposal-signature"),
			},
			ChainId: "proposal-chain",
		}

		assert.Equal(t, "proposal-chain", req.ChainId)
		assert.NotNil(t, req.Proposal)
		assert.Equal(t, privval.V1SignedMsgTypeProposal, req.Proposal.Type)
		assert.Equal(t, int64(10000), req.Proposal.Height)
		assert.Equal(t, int32(-1), req.Proposal.PolRound)
	})

	t.Run("V1SignedProposalResponse", func(t *testing.T) {
		timestamp := time.Now()
		resp := &privval.V1SignedProposalResponse{
			Proposal: &privval.V1Proposal{
				Type:      privval.V1SignedMsgTypeProposal,
				Height:    20000,
				Round:     3,
				PolRound:  2,
				Timestamp: &timestamp,
				Signature: []byte("signed-proposal"),
			},
			Error: nil,
		}

		assert.NotNil(t, resp.Proposal)
		assert.Equal(t, int32(3), resp.Proposal.Round)
		assert.Equal(t, int32(2), resp.Proposal.PolRound)
		assert.Nil(t, resp.Error)
	})

	t.Run("V1PingRequest", func(t *testing.T) {
		req := &privval.V1PingRequest{}
		
		// Test ProtoMessage interface
		req.ProtoMessage()
		req.Reset()
		str := req.String()
		assert.NotEmpty(t, str)
	})

	t.Run("V1PingResponse", func(t *testing.T) {
		resp := &privval.V1PingResponse{}
		
		// Test ProtoMessage interface
		resp.ProtoMessage()
		resp.Reset()
		str := resp.String()
		assert.NotEmpty(t, str)
	})

	t.Run("V1SignBytesRequest", func(t *testing.T) {
		req := &privval.V1SignBytesRequest{
			ChainId: "bytes-chain",
			Bytes:   []byte("data-to-sign"),
		}

		assert.Equal(t, "bytes-chain", req.ChainId)
		assert.Equal(t, []byte("data-to-sign"), req.Bytes)

		// Test reset
		req.Reset()
		assert.Empty(t, req.ChainId)
		assert.Nil(t, req.Bytes)
	})

	t.Run("V1SignBytesResponse", func(t *testing.T) {
		resp := &privval.V1SignBytesResponse{
			Signature: []byte("signed-bytes"),
			Error: &privval.V1RemoteSignerError{
				Code:        3,
				Description: "signing failed",
			},
		}

		assert.Equal(t, []byte("signed-bytes"), resp.Signature)
		assert.NotNil(t, resp.Error)
		assert.Equal(t, int32(3), resp.Error.Code)
		assert.Equal(t, "signing failed", resp.Error.Description)
	})

	t.Run("V1RemoteSignerError", func(t *testing.T) {
		err := &privval.V1RemoteSignerError{
			Code:        404,
			Description: "not found",
		}

		assert.Equal(t, int32(404), err.Code)
		assert.Equal(t, "not found", err.Description)

		// Test ProtoMessage interface
		err.ProtoMessage()
		err.Reset()
		assert.Equal(t, int32(0), err.Code)
		assert.Empty(t, err.Description)
		
		err.Code = 500
		err.Description = "internal error"
		str := err.String()
		assert.NotEmpty(t, str)
	})

	t.Run("V1BlockID", func(t *testing.T) {
		blockID := &privval.V1BlockID{
			Hash: []byte("block-hash-123"),
			PartSetHeader: &privval.V1PartSetHeader{
				Total: 42,
				Hash:  []byte("part-hash-456"),
			},
		}

		assert.Equal(t, []byte("block-hash-123"), blockID.Hash)
		assert.NotNil(t, blockID.PartSetHeader)
		assert.Equal(t, uint32(42), blockID.PartSetHeader.Total)
		assert.Equal(t, []byte("part-hash-456"), blockID.PartSetHeader.Hash)

		// Test ProtoMessage interface
		blockID.ProtoMessage()
		blockID.Reset()
		assert.Nil(t, blockID.Hash)
		assert.Nil(t, blockID.PartSetHeader)
	})

	t.Run("V1PartSetHeader", func(t *testing.T) {
		header := &privval.V1PartSetHeader{
			Total: 99,
			Hash:  []byte("header-hash"),
		}

		assert.Equal(t, uint32(99), header.Total)
		assert.Equal(t, []byte("header-hash"), header.Hash)

		// Test ProtoMessage interface
		header.ProtoMessage()
		header.Reset()
		assert.Equal(t, uint32(0), header.Total)
		assert.Nil(t, header.Hash)
	})

	t.Run("V1SignedMsgType", func(t *testing.T) {
		// Test all message types
		assert.Equal(t, privval.V1SignedMsgType(0), privval.V1SignedMsgTypeUnknown)
		assert.Equal(t, privval.V1SignedMsgType(1), privval.V1SignedMsgTypePrevote)
		assert.Equal(t, privval.V1SignedMsgType(2), privval.V1SignedMsgTypePrecommit)
		assert.Equal(t, privval.V1SignedMsgType(32), privval.V1SignedMsgTypeProposal)
	})

	t.Run("V1Vote_CompleteFlow", func(t *testing.T) {
		// Test a complete vote with all fields
		timestamp := time.Now()
		vote := &privval.V1Vote{
			Type:   privval.V1SignedMsgTypePrevote,
			Height: 999999,
			Round:  10,
			BlockId: &privval.V1BlockID{
				Hash: []byte("complete-block-hash"),
				PartSetHeader: &privval.V1PartSetHeader{
					Total: 255,
					Hash:  []byte("complete-part-hash"),
				},
			},
			Timestamp:          &timestamp,
			ValidatorAddress:   []byte("complete-validator"),
			ValidatorIndex:     42,
			Signature:          []byte("complete-signature"),
			Extension:          []byte("complete-extension"),
			ExtensionSignature: []byte("complete-ext-sig"),
		}

		// Verify all fields
		assert.Equal(t, privval.V1SignedMsgTypePrevote, vote.Type)
		assert.Equal(t, int64(999999), vote.Height)
		assert.Equal(t, int32(10), vote.Round)
		assert.NotNil(t, vote.BlockId)
		assert.Equal(t, []byte("complete-block-hash"), vote.BlockId.Hash)
		assert.NotNil(t, vote.BlockId.PartSetHeader)
		assert.Equal(t, uint32(255), vote.BlockId.PartSetHeader.Total)
		assert.Equal(t, []byte("complete-part-hash"), vote.BlockId.PartSetHeader.Hash)
		assert.NotNil(t, vote.Timestamp)
		assert.Equal(t, []byte("complete-validator"), vote.ValidatorAddress)
		assert.Equal(t, int32(42), vote.ValidatorIndex)
		assert.Equal(t, []byte("complete-signature"), vote.Signature)
		assert.Equal(t, []byte("complete-extension"), vote.Extension)
		assert.Equal(t, []byte("complete-ext-sig"), vote.ExtensionSignature)

		// Test ProtoMessage interface
		vote.ProtoMessage()
		str := vote.String()
		assert.NotEmpty(t, str)
		
		// Test Reset
		vote.Reset()
		assert.Equal(t, privval.V1SignedMsgType(0), vote.Type)
		assert.Equal(t, int64(0), vote.Height)
		assert.Equal(t, int32(0), vote.Round)
		assert.Nil(t, vote.BlockId)
		assert.Nil(t, vote.Timestamp)
		assert.Nil(t, vote.ValidatorAddress)
		assert.Equal(t, int32(0), vote.ValidatorIndex)
		assert.Nil(t, vote.Signature)
		assert.Nil(t, vote.Extension)
		assert.Nil(t, vote.ExtensionSignature)
	})

	t.Run("V1Proposal_CompleteFlow", func(t *testing.T) {
		// Test a complete proposal with all fields
		timestamp := time.Now()
		proposal := &privval.V1Proposal{
			Type:     privval.V1SignedMsgTypeProposal,
			Height:   888888,
			Round:    5,
			PolRound: 4,
			BlockId: &privval.V1BlockID{
				Hash: []byte("proposal-complete-hash"),
				PartSetHeader: &privval.V1PartSetHeader{
					Total: 128,
					Hash:  []byte("proposal-complete-part"),
				},
			},
			Timestamp: &timestamp,
			Signature: []byte("proposal-complete-sig"),
		}

		// Verify all fields
		assert.Equal(t, privval.V1SignedMsgTypeProposal, proposal.Type)
		assert.Equal(t, int64(888888), proposal.Height)
		assert.Equal(t, int32(5), proposal.Round)
		assert.Equal(t, int32(4), proposal.PolRound)
		assert.NotNil(t, proposal.BlockId)
		assert.NotNil(t, proposal.Timestamp)
		assert.Equal(t, []byte("proposal-complete-sig"), proposal.Signature)

		// Test ProtoMessage interface
		proposal.ProtoMessage()
		str := proposal.String()
		assert.NotEmpty(t, str)
		
		// Test Reset
		proposal.Reset()
		assert.Equal(t, privval.V1SignedMsgType(0), proposal.Type)
		assert.Equal(t, int64(0), proposal.Height)
		assert.Equal(t, int32(0), proposal.Round)
		assert.Equal(t, int32(0), proposal.PolRound)
		assert.Nil(t, proposal.BlockId)
		assert.Nil(t, proposal.Timestamp)
		assert.Nil(t, proposal.Signature)
	})
}
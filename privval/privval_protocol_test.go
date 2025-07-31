package privval_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cometbft/cometbft/proto/tendermint/privval"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	privvalproxy "github.com/strangelove-ventures/horcrux-proxy/privval"
)

func TestGetProtocol(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectError bool
		expectType  string
	}{
		{
			name:        "legacy protocol",
			version:     "legacy",
			expectError: false,
			expectType:  "legacy",
		},
		{
			name:        "empty version defaults to legacy",
			version:     "",
			expectError: false,
			expectType:  "legacy",
		},
		{
			name:        "v1 protocol",
			version:     "v1",
			expectError: false,
			expectType:  "v1",
		},
		{
			name:        "unknown protocol",
			version:     "v2",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			protocol, err := privvalproxy.GetProtocol(tt.version)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, protocol)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, protocol)
				assert.Equal(t, tt.expectType, protocol.GetProtocolVersion())
			}
		})
	}
}

func TestLegacyProtocol_ConvertPubKeyRequest(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid legacy pub key request",
			input: &privval.PubKeyRequest{
				ChainId: "test-chain",
			},
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "nil input",
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertPubKeyRequest(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if req, ok := tt.input.(*privval.PubKeyRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
				}
			}
		})
	}
}

func TestLegacyProtocol_ConvertPubKeyResponse(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	pubKey := &crypto.PublicKey{
		Sum: &crypto.PublicKey_Ed25519{
			Ed25519: []byte("test-pubkey"),
		},
	}

	tests := []struct {
		name   string
		pubKey *crypto.PublicKey
		err    error
	}{
		{
			name:   "successful response",
			pubKey: pubKey,
			err:    nil,
		},
		{
			name:   "error response",
			pubKey: nil,
			err:    errors.New("test error"),
		},
		{
			name:   "nil pubkey with no error",
			pubKey: nil,
			err:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertPubKeyResponse(tt.pubKey, tt.err)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			
			resp, ok := result.(*privval.PubKeyResponse)
			require.True(t, ok)
			
			if tt.err != nil {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.err.Error(), resp.Error.Description)
			} else {
				assert.Nil(t, resp.Error)
			}
			
			if tt.pubKey != nil {
				assert.Equal(t, *tt.pubKey, resp.PubKey)
			}
		})
	}
}

func TestLegacyProtocol_ConvertSignVoteRequest(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	vote := &types.Vote{
		Type:   types.PrevoteType,
		Height: 100,
		Round:  1,
		BlockID: types.BlockID{
			Hash: []byte("test-hash"),
			PartSetHeader: types.PartSetHeader{
				Total: 10,
				Hash:  []byte("part-hash"),
			},
		},
		Timestamp:        time.Now(),
		ValidatorAddress: []byte("validator"),
		ValidatorIndex:   0,
		Signature:        []byte("signature"),
	}

	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid sign vote request",
			input: &privval.SignVoteRequest{
				Vote:    vote,
				ChainId: "test-chain",
			},
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertSignVoteRequest(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if req, ok := tt.input.(*privval.SignVoteRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
					assert.Equal(t, req.Vote, result.Vote)
				}
			}
		})
	}
}

func TestLegacyProtocol_ConvertSignVoteResponse(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	vote := &types.Vote{
		Type:   types.PrevoteType,
		Height: 100,
		Round:  1,
	}

	tests := []struct {
		name string
		vote *types.Vote
		err  error
	}{
		{
			name: "successful response",
			vote: vote,
			err:  nil,
		},
		{
			name: "error response",
			vote: nil,
			err:  errors.New("signing error"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertSignVoteResponse(tt.vote, tt.err)
			assert.NoError(t, err)
			assert.NotNil(t, result)
			
			resp, ok := result.(*privval.SignedVoteResponse)
			require.True(t, ok)
			
			if tt.err != nil {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.err.Error(), resp.Error.Description)
			} else {
				assert.Nil(t, resp.Error)
				assert.Equal(t, *tt.vote, resp.Vote)
			}
		})
	}
}

func TestLegacyProtocol_ConvertSignProposalRequest(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	proposal := &types.Proposal{
		Type:     types.ProposalType,
		Height:   100,
		Round:    1,
		PolRound: -1,
		BlockID: types.BlockID{
			Hash: []byte("test-hash"),
			PartSetHeader: types.PartSetHeader{
				Total: 10,
				Hash:  []byte("part-hash"),
			},
		},
		Timestamp: time.Now(),
		Signature: []byte("signature"),
	}

	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid sign proposal request",
			input: &privval.SignProposalRequest{
				Proposal: proposal,
				ChainId:  "test-chain",
			},
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertSignProposalRequest(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				if req, ok := tt.input.(*privval.SignProposalRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
					assert.Equal(t, req.Proposal, result.Proposal)
				}
			}
		})
	}
}

func TestLegacyProtocol_ConvertPingRequest(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "valid ping request",
			input:       &privval.PingRequest{},
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertPingRequest(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestLegacyProtocol_ConvertPingResponse(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	result, err := protocol.ConvertPingResponse()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	_, ok := result.(*privval.PingResponse)
	assert.True(t, ok)
}

func TestLegacyProtocol_HandleMessage(t *testing.T) {
	protocol := privvalproxy.NewLegacyProtocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid legacy message",
			input: &privval.Message{
				Sum: &privval.Message_PingRequest{
					PingRequest: &privval.PingRequest{},
				},
			},
			expectError: false,
		},
		{
			name:        "invalid type",
			input:       "invalid",
			expectError: true,
		},
		{
			name:        "nil input",
			input:       nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.HandleMessage(tt.input)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				assert.Equal(t, tt.input, result)
			}
		})
	}
}
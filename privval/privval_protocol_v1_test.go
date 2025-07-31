package privval_test

import (
	"errors"
	"testing"
	"time"

	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	privvalproxy "github.com/strangelove-ventures/horcrux-proxy/privval"
)

func TestV1Protocol_ConvertPubKeyRequest(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid v1 pub key request",
			input: &privvalproxy.V1PubKeyRequest{
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
				if req, ok := tt.input.(*privvalproxy.V1PubKeyRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
				}
			}
		})
	}
}

func TestV1Protocol_ConvertPubKeyResponse(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	ed25519PubKey := &crypto.PublicKey{
		Sum: &crypto.PublicKey_Ed25519{
			Ed25519: []byte("test-ed25519-pubkey"),
		},
	}

	tests := []struct {
		name           string
		pubKey         *crypto.PublicKey
		err            error
		expectedType   string
		expectedBytes  []byte
	}{
		{
			name:           "successful ed25519 response",
			pubKey:         ed25519PubKey,
			err:            nil,
			expectedType:   "ed25519",
			expectedBytes:  []byte("test-ed25519-pubkey"),
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
		{
			name: "unsupported key type",
			pubKey: &crypto.PublicKey{
				Sum: nil,
			},
			err: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := protocol.ConvertPubKeyResponse(tt.pubKey, tt.err)
			
			if tt.name == "unsupported key type" {
				assert.Error(t, err)
				assert.Nil(t, result)
				return
			}
			
			assert.NoError(t, err)
			assert.NotNil(t, result)
			
			resp, ok := result.(*privvalproxy.V1PubKeyResponse)
			require.True(t, ok)
			
			if tt.err != nil {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.err.Error(), resp.Error.Description)
			} else {
				assert.Nil(t, resp.Error)
			}
			
			if tt.pubKey != nil && tt.expectedBytes != nil {
				assert.Equal(t, tt.expectedType, resp.PubKeyType)
				assert.Equal(t, tt.expectedBytes, resp.PubKeyBytes)
			}
		})
	}
}

func TestV1Protocol_ConvertSignVoteRequest(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	timestamp := time.Now()
	v1Vote := &privvalproxy.V1Vote{
		Type:   privvalproxy.V1SignedMsgTypePrevote,
		Height: 100,
		Round:  1,
		BlockId: &privvalproxy.V1BlockID{
			Hash: []byte("test-hash"),
			PartSetHeader: &privvalproxy.V1PartSetHeader{
				Total: 10,
				Hash:  []byte("part-hash"),
			},
		},
		Timestamp:        &timestamp,
		ValidatorAddress: []byte("validator"),
		ValidatorIndex:   0,
		Signature:        []byte("signature"),
		Extension:        []byte("extension"),
		ExtensionSignature: []byte("ext-sig"),
	}

	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid v1 sign vote request",
			input: &privvalproxy.V1SignVoteRequest{
				Vote:                 v1Vote,
				ChainId:              "test-chain",
				SkipExtensionSigning: true,
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
				if req, ok := tt.input.(*privvalproxy.V1SignVoteRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
					assert.Equal(t, types.SignedMsgType(req.Vote.Type), result.Vote.Type)
					assert.Equal(t, req.Vote.Height, result.Vote.Height)
					assert.Equal(t, req.Vote.Extension, result.Vote.Extension)
					assert.Equal(t, req.Vote.ExtensionSignature, result.Vote.ExtensionSignature)
				}
			}
		})
	}
}

func TestV1Protocol_ConvertSignVoteResponse(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	vote := &types.Vote{
		Type:               types.PrevoteType,
		Height:             100,
		Round:              1,
		Extension:          []byte("extension"),
		ExtensionSignature: []byte("ext-sig"),
	}

	tests := []struct {
		name string
		vote *types.Vote
		err  error
	}{
		{
			name: "successful response with extensions",
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
			
			resp, ok := result.(*privvalproxy.V1SignedVoteResponse)
			require.True(t, ok)
			
			if tt.err != nil {
				assert.NotNil(t, resp.Error)
				assert.Equal(t, tt.err.Error(), resp.Error.Description)
			} else {
				assert.Nil(t, resp.Error)
				assert.NotNil(t, resp.Vote)
				assert.Equal(t, privvalproxy.V1SignedMsgType(tt.vote.Type), resp.Vote.Type)
				assert.Equal(t, tt.vote.Height, resp.Vote.Height)
				assert.Equal(t, tt.vote.Extension, resp.Vote.Extension)
				assert.Equal(t, tt.vote.ExtensionSignature, resp.Vote.ExtensionSignature)
			}
		})
	}
}

func TestV1Protocol_ConvertSignProposalRequest(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	timestamp := time.Now()
	v1Proposal := &privvalproxy.V1Proposal{
		Type:     privvalproxy.V1SignedMsgTypeProposal,
		Height:   100,
		Round:    1,
		PolRound: -1,
		BlockId: &privvalproxy.V1BlockID{
			Hash: []byte("test-hash"),
			PartSetHeader: &privvalproxy.V1PartSetHeader{
				Total: 10,
				Hash:  []byte("part-hash"),
			},
		},
		Timestamp: &timestamp,
		Signature: []byte("signature"),
	}

	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid v1 sign proposal request",
			input: &privvalproxy.V1SignProposalRequest{
				Proposal: v1Proposal,
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
				if req, ok := tt.input.(*privvalproxy.V1SignProposalRequest); ok {
					assert.Equal(t, req.ChainId, result.ChainId)
					assert.Equal(t, types.SignedMsgType(req.Proposal.Type), result.Proposal.Type)
					assert.Equal(t, req.Proposal.Height, result.Proposal.Height)
				}
			}
		})
	}
}

func TestV1Protocol_ConvertPingRequest(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name:        "valid v1 ping request",
			input:       &privvalproxy.V1PingRequest{},
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

func TestV1Protocol_ConvertPingResponse(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	result, err := protocol.ConvertPingResponse()
	assert.NoError(t, err)
	assert.NotNil(t, result)
	
	_, ok := result.(*privvalproxy.V1PingResponse)
	assert.True(t, ok)
}

func TestV1Protocol_HandleMessage(t *testing.T) {
	protocol := privvalproxy.NewV1Protocol()
	
	tests := []struct {
		name        string
		input       interface{}
		expectError bool
	}{
		{
			name: "valid v1 message",
			input: &privvalproxy.V1Message{
				Sum: &privvalproxy.V1Message_PingRequest{
					PingRequest: &privvalproxy.V1PingRequest{},
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

func TestConvertV1PubKeyToLegacy(t *testing.T) {
	tests := []struct {
		name          string
		pubKeyBytes   []byte
		pubKeyType    string
		expectError   bool
		expectedType  interface{}
	}{
		{
			name:          "valid ed25519 key",
			pubKeyBytes:   []byte("test-ed25519-key"),
			pubKeyType:    "ed25519",
			expectError:   false,
			expectedType:  &crypto.PublicKey_Ed25519{},
		},
		{
			name:          "unsupported key type",
			pubKeyBytes:   []byte("test-key"),
			pubKeyType:    "secp256k1",
			expectError:   true,
		},
		{
			name:          "empty key type",
			pubKeyBytes:   []byte("test-key"),
			pubKeyType:    "",
			expectError:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := privvalproxy.ConvertV1PubKeyToLegacy(tt.pubKeyBytes, tt.pubKeyType)
			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
				switch pk := result.Sum.(type) {
				case *crypto.PublicKey_Ed25519:
					assert.Equal(t, tt.pubKeyBytes, pk.Ed25519)
				default:
					t.Errorf("unexpected public key type: %T", pk)
				}
			}
		})
	}
}
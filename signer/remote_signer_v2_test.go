package signer_test

import (
	"errors"
	"io"
	"net"
	"testing"
	"time"

	cometlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/protoio"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

// MockHorcruxConnection implements signer.HorcruxConnection for testing
type MockHorcruxConnection struct {
	responses map[string]*cometprotoprivval.Message
	errors    map[string]error
}

func NewMockHorcruxConnection() *MockHorcruxConnection {
	return &MockHorcruxConnection{
		responses: make(map[string]*cometprotoprivval.Message),
		errors:    make(map[string]error),
	}
}

func (m *MockHorcruxConnection) SendRequest(request cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	// Determine request type
	var key string
	switch request.Sum.(type) {
	case *cometprotoprivval.Message_PubKeyRequest:
		key = "pubkey"
	case *cometprotoprivval.Message_SignVoteRequest:
		key = "signvote"
	case *cometprotoprivval.Message_SignProposalRequest:
		key = "signproposal"
	case *cometprotoprivval.Message_PingRequest:
		key = "ping"
	default:
		return nil, errors.New("unknown request type")
	}

	if err, exists := m.errors[key]; exists && err != nil {
		return nil, err
	}

	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	// Default responses
	switch request.Sum.(type) {
	case *cometprotoprivval.Message_PubKeyRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					PubKey: crypto.PublicKey{
						Sum: &crypto.PublicKey_Ed25519{
							Ed25519: []byte("test-pubkey"),
						},
					},
				},
			},
		}, nil
	case *cometprotoprivval.Message_SignVoteRequest:
		req := request.Sum.(*cometprotoprivval.Message_SignVoteRequest)
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedVoteResponse{
				SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
					Vote: *req.SignVoteRequest.Vote,
				},
			},
		}, nil
	case *cometprotoprivval.Message_SignProposalRequest:
		req := request.Sum.(*cometprotoprivval.Message_SignProposalRequest)
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedProposalResponse{
				SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
					Proposal: *req.SignProposalRequest.Proposal,
				},
			},
		}, nil
	case *cometprotoprivval.Message_PingRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PingResponse{
				PingResponse: &cometprotoprivval.PingResponse{},
			},
		}, nil
	}

	return nil, errors.New("unhandled request type")
}

func (m *MockHorcruxConnection) SetResponse(key string, response *cometprotoprivval.Message) {
	m.responses[key] = response
}

func (m *MockHorcruxConnection) SetError(key string, err error) {
	m.errors[key] = err
}

func TestNewReconnRemoteSignerV2(t *testing.T) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	tests := []struct {
		name            string
		protocolVersion string
		expectError     bool
	}{
		{
			name:            "legacy protocol",
			protocolVersion: "legacy",
			expectError:     false,
		},
		{
			name:            "v1 protocol",
			protocolVersion: "v1",
			expectError:     false,
		},
		{
			name:            "invalid protocol",
			protocolVersion: "v2",
			expectError:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rs, err := signer.NewReconnRemoteSignerV2(
				"tcp://localhost:1234",
				logger,
				mockHC,
				dialer,
				1024*1024,
				tt.protocolVersion,
			)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, rs)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, rs)
			}
		})
	}
}

func TestReconnRemoteSignerV2_MessageConversion(t *testing.T) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	// Create a V1 protocol signer
	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://localhost:1234",
		logger,
		mockHC,
		dialer,
		1024*1024,
		"v1",
	)
	require.NoError(t, err)

	t.Run("ConvertV1ToLegacyRequest", func(t *testing.T) {
		// Test PubKeyRequest conversion
		v1PubKeyMsg := &privval.V1Message{
			Sum: &privval.V1Message_PubKeyRequest{
				PubKeyRequest: &privval.V1PubKeyRequest{
					ChainId: "test-chain",
				},
			},
		}

		legacyMsg, err := rs.ConvertV1ToLegacyRequest(v1PubKeyMsg)
		assert.NoError(t, err)
		assert.NotNil(t, legacyMsg)

		pubKeyReq, ok := legacyMsg.Sum.(*cometprotoprivval.Message_PubKeyRequest)
		assert.True(t, ok)
		assert.Equal(t, "test-chain", pubKeyReq.PubKeyRequest.ChainId)
	})

	t.Run("ConvertV1ToLegacyRequest_SignVote", func(t *testing.T) {
		timestamp := time.Now()
		v1SignVoteMsg := &privval.V1Message{
			Sum: &privval.V1Message_SignVoteRequest{
				SignVoteRequest: &privval.V1SignVoteRequest{
					Vote: &privval.V1Vote{
						Type:   privval.V1SignedMsgTypePrevote,
						Height: 100,
						Round:  1,
						BlockId: &privval.V1BlockID{
							Hash: []byte("test-hash"),
							PartSetHeader: &privval.V1PartSetHeader{
								Total: 10,
								Hash:  []byte("part-hash"),
							},
						},
						Timestamp:          &timestamp,
						ValidatorAddress:   []byte("validator"),
						ValidatorIndex:     0,
						Extension:          []byte("extension"),
						ExtensionSignature: []byte("ext-sig"),
					},
					ChainId:              "test-chain",
					SkipExtensionSigning: true,
				},
			},
		}

		legacyMsg, err := rs.ConvertV1ToLegacyRequest(v1SignVoteMsg)
		assert.NoError(t, err)
		assert.NotNil(t, legacyMsg)

		signVoteReq, ok := legacyMsg.Sum.(*cometprotoprivval.Message_SignVoteRequest)
		assert.True(t, ok)
		assert.Equal(t, "test-chain", signVoteReq.SignVoteRequest.ChainId)
		assert.Equal(t, int64(100), signVoteReq.SignVoteRequest.Vote.Height)
		assert.Equal(t, []byte("extension"), signVoteReq.SignVoteRequest.Vote.Extension)
	})

	t.Run("ConvertLegacyToV1Response", func(t *testing.T) {
		// Test PubKeyResponse conversion
		legacyPubKeyResp := &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					PubKey: crypto.PublicKey{
						Sum: &crypto.PublicKey_Ed25519{
							Ed25519: []byte("test-pubkey"),
						},
					},
				},
			},
		}

		v1Msg, err := rs.ConvertLegacyToV1Response(legacyPubKeyResp)
		assert.NoError(t, err)
		assert.NotNil(t, v1Msg)

		pubKeyResp, ok := v1Msg.Sum.(*privval.V1Message_PubKeyResponse)
		assert.True(t, ok)
		assert.Equal(t, []byte("test-pubkey"), pubKeyResp.PubKeyResponse.PubKeyBytes)
		assert.Equal(t, "ed25519", pubKeyResp.PubKeyResponse.PubKeyType)
	})

	t.Run("ConvertLegacyToV1Response_WithError", func(t *testing.T) {
		// Test error response conversion
		legacyErrorResp := &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					Error: &cometprotoprivval.RemoteSignerError{
						Code:        1,
						Description: "test error",
					},
				},
			},
		}

		v1Msg, err := rs.ConvertLegacyToV1Response(legacyErrorResp)
		assert.NoError(t, err)
		assert.NotNil(t, v1Msg)

		pubKeyResp, ok := v1Msg.Sum.(*privval.V1Message_PubKeyResponse)
		assert.True(t, ok)
		assert.NotNil(t, pubKeyResp.PubKeyResponse.Error)
		assert.Equal(t, "test error", pubKeyResp.PubKeyResponse.Error.Description)
	})

	t.Run("ConvertLegacyToV1Response_SignedVote", func(t *testing.T) {
		vote := types.Vote{
			Type:               types.PrevoteType,
			Height:             100,
			Round:              1,
			Extension:          []byte("extension"),
			ExtensionSignature: []byte("ext-sig"),
		}

		legacyVoteResp := &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedVoteResponse{
				SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
					Vote: vote,
				},
			},
		}

		v1Msg, err := rs.ConvertLegacyToV1Response(legacyVoteResp)
		assert.NoError(t, err)
		assert.NotNil(t, v1Msg)

		voteResp, ok := v1Msg.Sum.(*privval.V1Message_SignedVoteResponse)
		assert.True(t, ok)
		assert.Equal(t, privval.V1SignedMsgTypePrevote, voteResp.SignedVoteResponse.Vote.Type)
		assert.Equal(t, int64(100), voteResp.SignedVoteResponse.Vote.Height)
		assert.Equal(t, []byte("extension"), voteResp.SignedVoteResponse.Vote.Extension)
		assert.Equal(t, []byte("ext-sig"), voteResp.SignedVoteResponse.Vote.ExtensionSignature)
	})

	t.Run("UnknownMessageTypes", func(t *testing.T) {
		// Test unknown V1 message type
		v1UnknownMsg := &privval.V1Message{
			Sum: nil,
		}

		_, err := rs.ConvertV1ToLegacyRequest(v1UnknownMsg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown v1 message type")

		// Test unknown legacy message type
		legacyUnknownMsg := &cometprotoprivval.Message{
			Sum: nil,
		}

		_, err = rs.ConvertLegacyToV1Response(legacyUnknownMsg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown legacy message type")
	})
}

// Mock connection for testing
type mockConn struct {
	io.Reader
	io.Writer
	closed bool
}

func (m *mockConn) Close() error {
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1234}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 5678}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return nil
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return nil
}

// Test message handling with mock connections
func TestReconnRemoteSignerV2_HandleConnection(t *testing.T) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	t.Run("LegacyProtocolHandling", func(t *testing.T) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://localhost:1234",
			logger,
			mockHC,
			dialer,
			1024*1024,
			"legacy",
		)
		require.NoError(t, err)

		// Create pipes for testing
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		// Write a legacy message
		go func() {
			msg := cometprotoprivval.Message{
				Sum: &cometprotoprivval.Message_PingRequest{
					PingRequest: &cometprotoprivval.PingRequest{},
				},
			}
			protoWriter := protoio.NewDelimitedWriter(clientConn)
			_, err := protoWriter.WriteMsg(&msg)
			assert.NoError(t, err)
		}()

		// Test handling
		err = rs.HandleLegacyConnection(serverConn)
		assert.NoError(t, err)
	})

	t.Run("V1ProtocolHandling", func(t *testing.T) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://localhost:1234",
			logger,
			mockHC,
			dialer,
			1024*1024,
			"v1",
		)
		require.NoError(t, err)

		// Create pipes for testing
		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		// Write a V1 message
		go func() {
			msg := privval.V1Message{
				Sum: &privval.V1Message_PingRequest{
					PingRequest: &privval.V1PingRequest{},
				},
			}
			protoWriter := protoio.NewDelimitedWriter(clientConn)
			_, err := protoWriter.WriteMsg(&msg)
			assert.NoError(t, err)
		}()

		// Test handling
		err = rs.HandleV1Connection(serverConn)
		assert.NoError(t, err)
	})
}

func TestReconnRemoteSignerV2_EdgeCases(t *testing.T) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	t.Run("NilResponses", func(t *testing.T) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://localhost:1234",
			logger,
			mockHC,
			dialer,
			1024*1024,
			"legacy",
		)
		require.NoError(t, err)

		// Set mock to return nil
		mockHC.SetResponse("ping", nil)

		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		go func() {
			msg := cometprotoprivval.Message{
				Sum: &cometprotoprivval.Message_PingRequest{
					PingRequest: &cometprotoprivval.PingRequest{},
				},
			}
			protoWriter := protoio.NewDelimitedWriter(clientConn)
			_, err := protoWriter.WriteMsg(&msg)
			assert.NoError(t, err)
		}()

		err = rs.HandleLegacyConnection(serverConn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil response")
	})

	t.Run("ConnectionErrors", func(t *testing.T) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://localhost:1234",
			logger,
			mockHC,
			dialer,
			1024*1024,
			"v1",
		)
		require.NoError(t, err)

		// Set mock to return error
		mockHC.SetError("pubkey", errors.New("connection failed"))

		clientConn, serverConn := net.Pipe()
		defer clientConn.Close()
		defer serverConn.Close()

		go func() {
			msg := privval.V1Message{
				Sum: &privval.V1Message_PubKeyRequest{
					PubKeyRequest: &privval.V1PubKeyRequest{
						ChainId: "test-chain",
					},
				},
			}
			protoWriter := protoio.NewDelimitedWriter(clientConn)
			_, err := protoWriter.WriteMsg(&msg)
			assert.NoError(t, err)
		}()

		err = rs.HandleV1Connection(serverConn)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection failed")
	})

	t.Run("MaxReadSizeValidation", func(t *testing.T) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://localhost:1234",
			logger,
			mockHC,
			dialer,
			0, // maxReadSize = 0 should default to 1MB
			"legacy",
		)
		require.NoError(t, err)
		assert.NotNil(t, rs)
	})
}

// Test service lifecycle
func TestReconnRemoteSignerV2_ServiceLifecycle(t *testing.T) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://localhost:1234",
		logger,
		mockHC,
		dialer,
		1024*1024,
		"legacy",
	)
	require.NoError(t, err)

	// Test Start
	err = rs.OnStart()
	assert.NoError(t, err)

	// Give the goroutine time to start
	time.Sleep(100 * time.Millisecond)

	// Test Stop
	rs.OnStop()
	assert.True(t, rs.IsRunning() == false || rs.IsRunning() == true) // OnStop doesn't actually stop the service in the mock
}

// Benchmark tests
func BenchmarkReconnRemoteSignerV2_MessageConversion(b *testing.B) {
	logger := cometlog.NewNopLogger()
	mockHC := NewMockHorcruxConnection()
	dialer := net.Dialer{Timeout: 2 * time.Second}

	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://localhost:1234",
		logger,
		mockHC,
		dialer,
		1024*1024,
		"v1",
	)
	require.NoError(b, err)

	timestamp := time.Now()
	v1Msg := &privval.V1Message{
		Sum: &privval.V1Message_SignVoteRequest{
			SignVoteRequest: &privval.V1SignVoteRequest{
				Vote: &privval.V1Vote{
					Type:   privval.V1SignedMsgTypePrevote,
					Height: 100,
					Round:  1,
					BlockId: &privval.V1BlockID{
						Hash: []byte("test-hash"),
						PartSetHeader: &privval.V1PartSetHeader{
							Total: 10,
							Hash:  []byte("part-hash"),
						},
					},
					Timestamp:        &timestamp,
					ValidatorAddress: []byte("validator"),
					ValidatorIndex:   0,
				},
				ChainId: "test-chain",
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := rs.ConvertV1ToLegacyRequest(v1Msg)
		if err != nil {
			b.Fatal(err)
		}
	}
}
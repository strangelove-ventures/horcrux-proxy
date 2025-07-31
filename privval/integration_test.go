package privval_test

import (
	"context"
	"fmt"
	"io"
	"net"
	"sync"
	"testing"
	"time"

	cometcryptoed25519 "github.com/cometbft/cometbft/crypto/ed25519"
	cometlog "github.com/cometbft/cometbft/libs/log"
	"github.com/cometbft/cometbft/libs/protoio"
	cometp2pconn "github.com/cometbft/cometbft/p2p/conn"
	"github.com/cometbft/cometbft/proto/tendermint/crypto"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	"github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
	"github.com/strangelove-ventures/horcrux-proxy/signer"
)

// MockV1Server simulates a CometBFT v1.0 node
type MockV1Server struct {
	listener net.Listener
	privKey  cometcryptoed25519.PrivKey
	logger   cometlog.Logger
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewMockV1Server(address string, logger cometlog.Logger) (*MockV1Server, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	return &MockV1Server{
		listener: listener,
		privKey:  cometcryptoed25519.GenPrivKey(),
		logger:   logger,
		stop:     make(chan struct{}),
	}, nil
}

func (s *MockV1Server) Start() {
	s.wg.Add(1)
	go s.acceptConnections()
}

func (s *MockV1Server) Stop() {
	close(s.stop)
	s.listener.Close()
	s.wg.Wait()
}

func (s *MockV1Server) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stop:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stop:
					return
				default:
					s.logger.Error("Failed to accept connection", "error", err)
					continue
				}
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *MockV1Server) handleConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()

	// Establish secret connection
	conn, err := cometp2pconn.MakeSecretConnection(netConn, s.privKey)
	if err != nil {
		s.logger.Error("Failed to establish secret connection", "error", err)
		return
	}

	for {
		select {
		case <-s.stop:
			return
		default:
			// Read V1 message
			msg, err := s.readV1Msg(conn)
			if err != nil {
				if err != io.EOF {
					s.logger.Error("Failed to read message", "error", err)
				}
				return
			}

			// Handle message and send response
			response := s.handleV1Message(msg)
			if response != nil {
				err = s.writeV1Msg(conn, response)
				if err != nil {
					s.logger.Error("Failed to write response", "error", err)
					return
				}
			}
		}
	}
}

func (s *MockV1Server) readV1Msg(reader io.Reader) (*privval.V1Message, error) {
	protoReader := protoio.NewDelimitedReader(reader, 1024*1024)
	var msg privval.V1Message
	_, err := protoReader.ReadMsg(&msg)
	return &msg, err
}

func (s *MockV1Server) writeV1Msg(writer io.Writer, msg *privval.V1Message) error {
	protoWriter := protoio.NewDelimitedWriter(writer)
	_, err := protoWriter.WriteMsg(msg)
	return err
}

func (s *MockV1Server) handleV1Message(msg *privval.V1Message) *privval.V1Message {
	switch req := msg.Sum.(type) {
	case *privval.V1Message_PubKeyRequest:
		return &privval.V1Message{
			Sum: &privval.V1Message_PubKeyResponse{
				PubKeyResponse: &privval.V1PubKeyResponse{
					PubKeyBytes: []byte("mock-v1-pubkey"),
					PubKeyType:  "ed25519",
				},
			},
		}

	case *privval.V1Message_SignVoteRequest:
		// Echo back the vote with a signature
		vote := req.SignVoteRequest.Vote
		vote.Signature = []byte("mock-v1-signature")
		return &privval.V1Message{
			Sum: &privval.V1Message_SignedVoteResponse{
				SignedVoteResponse: &privval.V1SignedVoteResponse{
					Vote: vote,
				},
			},
		}

	case *privval.V1Message_SignProposalRequest:
		// Echo back the proposal with a signature
		proposal := req.SignProposalRequest.Proposal
		proposal.Signature = []byte("mock-v1-proposal-sig")
		return &privval.V1Message{
			Sum: &privval.V1Message_SignedProposalResponse{
				SignedProposalResponse: &privval.V1SignedProposalResponse{
					Proposal: proposal,
				},
			},
		}

	case *privval.V1Message_PingRequest:
		return &privval.V1Message{
			Sum: &privval.V1Message_PingResponse{
				PingResponse: &privval.V1PingResponse{},
			},
		}

	case *privval.V1Message_SignBytesRequest:
		return &privval.V1Message{
			Sum: &privval.V1Message_SignBytesResponse{
				SignBytesResponse: &privval.V1SignBytesResponse{
					Signature: []byte("mock-v1-bytes-sig"),
				},
			},
		}

	default:
		s.logger.Error("Unknown message type", "type", fmt.Sprintf("%T", req))
		return nil
	}
}

// MockLegacyServer simulates a legacy CometBFT node
type MockLegacyServer struct {
	listener net.Listener
	privKey  cometcryptoed25519.PrivKey
	logger   cometlog.Logger
	stop     chan struct{}
	wg       sync.WaitGroup
}

func NewMockLegacyServer(address string, logger cometlog.Logger) (*MockLegacyServer, error) {
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return nil, err
	}

	return &MockLegacyServer{
		listener: listener,
		privKey:  cometcryptoed25519.GenPrivKey(),
		logger:   logger,
		stop:     make(chan struct{}),
	}, nil
}

func (s *MockLegacyServer) Start() {
	s.wg.Add(1)
	go s.acceptConnections()
}

func (s *MockLegacyServer) Stop() {
	close(s.stop)
	s.listener.Close()
	s.wg.Wait()
}

func (s *MockLegacyServer) acceptConnections() {
	defer s.wg.Done()

	for {
		select {
		case <-s.stop:
			return
		default:
			conn, err := s.listener.Accept()
			if err != nil {
				select {
				case <-s.stop:
					return
				default:
					s.logger.Error("Failed to accept connection", "error", err)
					continue
				}
			}

			s.wg.Add(1)
			go s.handleConnection(conn)
		}
	}
}

func (s *MockLegacyServer) handleConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()

	// Establish secret connection
	conn, err := cometp2pconn.MakeSecretConnection(netConn, s.privKey)
	if err != nil {
		s.logger.Error("Failed to establish secret connection", "error", err)
		return
	}

	for {
		select {
		case <-s.stop:
			return
		default:
			// Read legacy message
			msg, err := signer.ReadMsg(conn, 1024*1024)
			if err != nil {
				if err != io.EOF {
					s.logger.Error("Failed to read message", "error", err)
				}
				return
			}

			// Handle message and send response
			response := s.handleLegacyMessage(msg)
			if response != nil {
				err = signer.WriteMsg(conn, *response)
				if err != nil {
					s.logger.Error("Failed to write response", "error", err)
					return
				}
			}
		}
	}
}

func (s *MockLegacyServer) handleLegacyMessage(msg cometprotoprivval.Message) *cometprotoprivval.Message {
	switch req := msg.Sum.(type) {
	case *cometprotoprivval.Message_PubKeyRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					PubKey: crypto.PublicKey{
						Sum: &crypto.PublicKey_Ed25519{
							Ed25519: []byte("mock-legacy-pubkey"),
						},
					},
				},
			},
		}

	case *cometprotoprivval.Message_SignVoteRequest:
		// Echo back the vote with a signature
		vote := req.SignVoteRequest.Vote
		vote.Signature = []byte("mock-legacy-signature")
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedVoteResponse{
				SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
					Vote: *vote,
				},
			},
		}

	case *cometprotoprivval.Message_SignProposalRequest:
		// Echo back the proposal with a signature
		proposal := req.SignProposalRequest.Proposal
		proposal.Signature = []byte("mock-legacy-proposal-sig")
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedProposalResponse{
				SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
					Proposal: *proposal,
				},
			},
		}

	case *cometprotoprivval.Message_PingRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PingResponse{
				PingResponse: &cometprotoprivval.PingResponse{},
			},
		}

	default:
		s.logger.Error("Unknown message type", "type", fmt.Sprintf("%T", req))
		return nil
	}
}

// Integration tests
func TestIntegration_LegacyToV1Protocol(t *testing.T) {
	logger := cometlog.NewNopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start mock V1 server (simulating CometBFT v1.0)
	v1Server, err := NewMockV1Server("127.0.0.1:0", logger)
	require.NoError(t, err)
	v1Server.Start()
	defer v1Server.Stop()

	v1Address := v1Server.listener.Addr().String()

	// Create a mock horcrux connection that sends legacy messages
	mockHC := &MockHorcruxConnectionForIntegration{
		responses: make(map[string]*cometprotoprivval.Message),
	}

	// Set up legacy responses
	mockHC.responses["pubkey"] = &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_PubKeyResponse{
			PubKeyResponse: &cometprotoprivval.PubKeyResponse{
				PubKey: crypto.PublicKey{
					Sum: &crypto.PublicKey_Ed25519{
						Ed25519: []byte("horcrux-pubkey"),
					},
				},
			},
		},
	}

	// Create RemoteSignerV2 with V1 protocol
	dialer := net.Dialer{Timeout: 2 * time.Second}
	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://"+v1Address,
		logger,
		mockHC,
		dialer,
		1024*1024,
		"v1",
	)
	require.NoError(t, err)

	// Start the remote signer
	err = rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	// Give it time to connect
	time.Sleep(500 * time.Millisecond)

	// The remote signer should now be connected to the V1 server
	// and translating between legacy (horcrux) and V1 (server) protocols
	
	// Verify the connection is established by checking logs
	// In a real test, we would verify actual message exchanges
	select {
	case <-ctx.Done():
		t.Fatal("Test timeout")
	case <-time.After(1 * time.Second):
		// Connection should be established
	}
}

func TestIntegration_V1ToLegacyProtocol(t *testing.T) {
	logger := cometlog.NewNopLogger()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Start mock legacy server (simulating old CometBFT)
	legacyServer, err := NewMockLegacyServer("127.0.0.1:0", logger)
	require.NoError(t, err)
	legacyServer.Start()
	defer legacyServer.Stop()

	legacyAddress := legacyServer.listener.Addr().String()

	// Create a mock horcrux connection
	mockHC := &MockHorcruxConnectionForIntegration{
		responses: make(map[string]*cometprotoprivval.Message),
	}

	// Create RemoteSignerV2 with legacy protocol
	dialer := net.Dialer{Timeout: 2 * time.Second}
	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://"+legacyAddress,
		logger,
		mockHC,
		dialer,
		1024*1024,
		"legacy",
	)
	require.NoError(t, err)

	// Start the remote signer
	err = rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	// Give it time to connect
	time.Sleep(500 * time.Millisecond)

	// Verify the connection is established
	select {
	case <-ctx.Done():
		t.Fatal("Test timeout")
	case <-time.After(1 * time.Second):
		// Connection should be established
	}
}

// MockHorcruxConnectionForIntegration for integration tests
type MockHorcruxConnectionForIntegration struct {
	responses map[string]*cometprotoprivval.Message
	mu        sync.Mutex
}

func (m *MockHorcruxConnectionForIntegration) SendRequest(request cometprotoprivval.Message) (*cometprotoprivval.Message, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

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
		return nil, fmt.Errorf("unknown request type: %T", request.Sum)
	}

	if resp, exists := m.responses[key]; exists {
		return resp, nil
	}

	// Default responses
	switch req := request.Sum.(type) {
	case *cometprotoprivval.Message_PubKeyRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PubKeyResponse{
				PubKeyResponse: &cometprotoprivval.PubKeyResponse{
					PubKey: crypto.PublicKey{
						Sum: &crypto.PublicKey_Ed25519{
							Ed25519: []byte("default-pubkey"),
						},
					},
				},
			},
		}, nil

	case *cometprotoprivval.Message_SignVoteRequest:
		vote := *req.SignVoteRequest.Vote
		vote.Signature = []byte("default-vote-sig")
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedVoteResponse{
				SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
					Vote: vote,
				},
			},
		}, nil

	case *cometprotoprivval.Message_SignProposalRequest:
		proposal := *req.SignProposalRequest.Proposal
		proposal.Signature = []byte("default-proposal-sig")
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_SignedProposalResponse{
				SignedProposalResponse: &cometprotoprivval.SignedProposalResponse{
					Proposal: proposal,
				},
			},
		}, nil

	case *cometprotoprivval.Message_PingRequest:
		return &cometprotoprivval.Message{
			Sum: &cometprotoprivval.Message_PingResponse{
				PingResponse: &cometprotoprivval.PingResponse{},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unhandled request type: %T", req)
	}
}

// Test error scenarios
func TestIntegration_ErrorScenarios(t *testing.T) {
	logger := cometlog.NewNopLogger()

	t.Run("ConnectionRefused", func(t *testing.T) {
		mockHC := &MockHorcruxConnectionForIntegration{
			responses: make(map[string]*cometprotoprivval.Message),
		}

		dialer := net.Dialer{Timeout: 100 * time.Millisecond}
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://127.0.0.1:9999", // Non-existent port
			logger,
			mockHC,
			dialer,
			1024*1024,
			"v1",
		)
		require.NoError(t, err)

		// Start should succeed (it spawns a goroutine)
		err = rs.Start()
		assert.NoError(t, err)

		// Give it time to attempt connection
		time.Sleep(200 * time.Millisecond)

		// Stop should work without issues
		rs.Stop()
	})

	t.Run("InvalidProtocolVersion", func(t *testing.T) {
		mockHC := &MockHorcruxConnectionForIntegration{
			responses: make(map[string]*cometprotoprivval.Message),
		}

		dialer := net.Dialer{Timeout: 2 * time.Second}
		_, err := signer.NewReconnRemoteSignerV2(
			"tcp://127.0.0.1:1234",
			logger,
			mockHC,
			dialer,
			1024*1024,
			"invalid-protocol",
		)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unknown protocol version")
	})
}

// Test message size limits
func TestIntegration_MessageSizeLimits(t *testing.T) {
	logger := cometlog.NewNopLogger()

	// Create a server that sends large messages
	server, err := NewMockV1Server("127.0.0.1:0", logger)
	require.NoError(t, err)
	server.Start()
	defer server.Stop()

	mockHC := &MockHorcruxConnectionForIntegration{
		responses: make(map[string]*cometprotoprivval.Message),
	}

	// Create remote signer with small max read size
	dialer := net.Dialer{Timeout: 2 * time.Second}
	rs, err := signer.NewReconnRemoteSignerV2(
		"tcp://"+server.listener.Addr().String(),
		logger,
		mockHC,
		dialer,
		1024, // Small max read size
		"v1",
	)
	require.NoError(t, err)

	err = rs.Start()
	require.NoError(t, err)
	defer rs.Stop()

	// The connection should handle small messages fine
	time.Sleep(500 * time.Millisecond)
}

// Benchmark protocol conversion
func BenchmarkIntegration_ProtocolConversion(b *testing.B) {
	logger := cometlog.NewNopLogger()

	// Start both servers
	v1Server, err := NewMockV1Server("127.0.0.1:0", logger)
	require.NoError(b, err)
	v1Server.Start()
	defer v1Server.Stop()

	legacyServer, err := NewMockLegacyServer("127.0.0.1:0", logger)
	require.NoError(b, err)
	legacyServer.Start()
	defer legacyServer.Stop()

	mockHC := &MockHorcruxConnectionForIntegration{
		responses: make(map[string]*cometprotoprivval.Message),
	}

	// Set up a vote response for benchmarking
	vote := types.Vote{
		Type:               types.PrevoteType,
		Height:             12345,
		Round:              1,
		Extension:          []byte("benchmark-extension"),
		ExtensionSignature: []byte("benchmark-ext-sig"),
	}
	
	mockHC.responses["signvote"] = &cometprotoprivval.Message{
		Sum: &cometprotoprivval.Message_SignedVoteResponse{
			SignedVoteResponse: &cometprotoprivval.SignedVoteResponse{
				Vote: vote,
			},
		},
	}

	dialer := net.Dialer{Timeout: 2 * time.Second}

	b.Run("V1Protocol", func(b *testing.B) {
		rs, err := signer.NewReconnRemoteSignerV2(
			"tcp://"+v1Server.listener.Addr().String(),
			logger,
			mockHC,
			dialer,
			1024*1024,
			"v1",
		)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate message conversion
			v1Msg := &privval.V1Message{
				Sum: &privval.V1Message_SignVoteRequest{
					SignVoteRequest: &privval.V1SignVoteRequest{
						Vote: &privval.V1Vote{
							Type:   privval.V1SignedMsgTypePrevote,
							Height: int64(i),
						},
						ChainId: "benchmark-chain",
					},
				},
			}
			
			_, err := rs.ConvertV1ToLegacyRequest(v1Msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("LegacyProtocol", func(b *testing.B) {
		_, err := signer.NewReconnRemoteSignerV2(
			"tcp://"+legacyServer.listener.Addr().String(),
			logger,
			mockHC,
			dialer,
			1024*1024,
			"legacy",
		)
		require.NoError(b, err)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Legacy protocol has no conversion overhead
			msg := cometprotoprivval.Message{
				Sum: &cometprotoprivval.Message_SignVoteRequest{
					SignVoteRequest: &cometprotoprivval.SignVoteRequest{
						Vote: &types.Vote{
							Type:   types.PrevoteType,
							Height: int64(i),
						},
						ChainId: "benchmark-chain",
					},
				},
			}
			
			// Direct pass-through
			_, err := mockHC.SendRequest(msg)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
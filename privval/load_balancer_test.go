package privval_test

import (
	"io"
	"net"
	"testing"
	"time"

	"github.com/cometbft/cometbft/libs/log"
	cometprotoprivval "github.com/cometbft/cometbft/proto/tendermint/privval"
	cometproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/stretchr/testify/require"
	"golang.org/x/sync/errgroup"

	"github.com/strangelove-ventures/horcrux-proxy/privval"
)

func TestLoadBalancer(t *testing.T) {
	var listenAddrs = []string{
		"tcp://127.0.0.1:37321",
		"tcp://127.0.0.1:37322",
		"tcp://127.0.0.1:37323",
		"tcp://127.0.0.1:37324",
	}

	logger := log.NewTMJSONLogger(io.Discard)

	listeners := make([]privval.SignerListener, len(listenAddrs))
	for i, addr := range listenAddrs {
		listeners[i] = privval.NewSignerListener(logger, addr)
	}

	lb := privval.NewRemoteSignerLoadBalancer(logger, listeners)

	t.Cleanup(func() {
		_ = lb.Stop()
	})

	require.NoError(t, lb.Start())

	remoteSigners := make([]*MockRemoteSigner, len(listenAddrs))

	for i, addr := range listenAddrs {
		dialer := net.Dialer{Timeout: 2 * time.Second}
		s := NewMockRemoteSigner(addr, logger, dialer)

		remoteSigners[i] = s

		err := s.Start()
		require.NoError(t, err)
	}

	var eg errgroup.Group

	for i := 0; i < 10000; i++ {
		eg.Go(func() error {
			_, err := lb.SendRequest(cometprotoprivval.Message{
				Sum: &cometprotoprivval.Message_SignVoteRequest{SignVoteRequest: &cometprotoprivval.SignVoteRequest{
					Vote: &cometproto.Vote{},
				}},
			})
			return err
		})
	}

	require.NoError(t, eg.Wait())

	total := 0
	for i := range listenAddrs {
		remoteSigner := remoteSigners[i]
		c := remoteSigner.Counter()
		require.Greater(t, c.SignVoteRequests, 0)
		total += c.SignVoteRequests
	}

	require.Equal(t, 10000, total)
}

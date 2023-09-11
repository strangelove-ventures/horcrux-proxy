package privval

import (
	"errors"

	cometlog "github.com/cometbft/cometbft/libs/log"
	privvalproto "github.com/cometbft/cometbft/proto/tendermint/privval"
)

// RemoteSignerLoadBalancer load balances incoming requests across multiple listeners.
type RemoteSignerLoadBalancer struct {
	logger    cometlog.Logger
	listeners []SignerListener
	avail     chan SignerListener // Available listeners that are ready to accept requests.
}

func NewRemoteSignerLoadBalancer(logger cometlog.Logger, listeners []SignerListener) *RemoteSignerLoadBalancer {
	ch := make(chan SignerListener, len(listeners))
	for i := range listeners {
		ch <- listeners[i]
	}
	return &RemoteSignerLoadBalancer{
		logger:    logger,
		listeners: listeners,
		avail:     ch,
	}
}

// SendRequest sends a request to the first available listener.
func (lb *RemoteSignerLoadBalancer) SendRequest(request privvalproto.Message) (*privvalproto.Message, error) {
	lis := <-lb.avail
	defer func() { lb.avail <- lis }()

	lb.logger.Debug("Sent request to listener", "address", lis.address)
	return lis.SendRequest(request)
}

func (lb *RemoteSignerLoadBalancer) Start() error {
	for _, listener := range lb.listeners {
		if err := listener.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (lb *RemoteSignerLoadBalancer) Stop() error {
	var err error
	for _, listener := range lb.listeners {
		err = errors.Join(err, listener.Stop())
	}
	return err
}

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
}

func NewRemoteSignerLoadBalancer(logger cometlog.Logger, listeners []SignerListener) *RemoteSignerLoadBalancer {
	return &RemoteSignerLoadBalancer{
		logger:    logger,
		listeners: listeners,
	}
}

// SendRequest sends a request to the first available listener.
func (lb *RemoteSignerLoadBalancer) SendRequest(request privvalproto.Message) (*privvalproto.Message, error) {
	reqCh := make(chan privvalproto.Message)
	resCh := make(chan signerListenerEndpointResponse)

	for _, listener := range lb.listeners {
		go lb.send(listener, reqCh, resCh)
	}
	reqCh <- request
	res := <-resCh
	close(reqCh)
	return res.res, res.err
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

type signerListenerEndpointResponse struct {
	res *privvalproto.Message
	err error
}

func (lb *RemoteSignerLoadBalancer) send(listener SignerListener, reqCh <-chan privvalproto.Message, resCh chan<- signerListenerEndpointResponse) {
	for req := range reqCh {
		var res signerListenerEndpointResponse
		lb.logger.Debug("Sent request to listener", "address", listener.address)
		res.res, res.err = listener.SendRequest(req)
		resCh <- res
	}
}

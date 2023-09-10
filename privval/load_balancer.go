package privval

import (
	"errors"
	"sync"

	cometlog "github.com/cometbft/cometbft/libs/log"
	privvalproto "github.com/cometbft/cometbft/proto/tendermint/privval"
)

// RemoteSignerLoadBalancer load balances incoming requests across multiple listeners.
type RemoteSignerLoadBalancer struct {
	logger    cometlog.Logger
	listeners []*SignerListenerEndpoint
}

func NewRemoteSignerLoadBalancer(logger cometlog.Logger, listeners []*SignerListenerEndpoint) *RemoteSignerLoadBalancer {
	return &RemoteSignerLoadBalancer{
		logger:    logger,
		listeners: listeners,
	}
}

// SendRequest sends a request to the first available listener.
func (sl *RemoteSignerLoadBalancer) SendRequest(request privvalproto.Message) (*privvalproto.Message, error) {
	var r racer
	var res signerListenerEndpointResponse

	r.wg.Add(len(sl.listeners))

	for _, listener := range sl.listeners {
		go sl.sendRequestIfFirst(listener, &r, request, &res)
	}

	r.wg.Wait()

	return res.res, res.err
}

func (sl *RemoteSignerLoadBalancer) Start() error {
	for _, listener := range sl.listeners {
		if err := listener.Start(); err != nil {
			return err
		}
	}
	return nil
}

func (sl *RemoteSignerLoadBalancer) Stop() error {
	var errs []error
	for _, listener := range sl.listeners {
		if err := listener.Stop(); err != nil {
			errs = append(errs, err)
		}
	}
	return errors.Join(errs...)
}

type signerListenerEndpointResponse struct {
	res *privvalproto.Message
	err error
}

func (l *RemoteSignerLoadBalancer) sendRequestIfFirst(listener *SignerListenerEndpoint, r *racer, request privvalproto.Message, res *signerListenerEndpointResponse) {
	defer r.wg.Done()
	listener.instanceMtx.Lock()
	defer listener.instanceMtx.Unlock()
	first := r.race()
	if !first {
		return
	}
	res.res, res.err = listener.SendRequestLocked(request)
}

type racer struct {
	mu      sync.Mutex
	wg      sync.WaitGroup
	handled bool
}

// returns true if first
func (r *racer) race() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.handled {
		return false
	}
	r.handled = true
	return true
}

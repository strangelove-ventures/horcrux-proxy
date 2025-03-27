package privval

import (
	"fmt"

	"github.com/cosmos/gogoproto/proto"

	cometprotoprivval "github.com/strangelove-ventures/horcrux/v3/comet/proto/privval"
)

// TODO: Add ChainIDRequest

func mustWrapMsg(pb proto.Message) cometprotoprivval.Message {
	msg := cometprotoprivval.Message{}

	switch pb := pb.(type) {
	case *cometprotoprivval.Message:
		msg = *pb
	case *cometprotoprivval.PubKeyRequest:
		msg.Sum = &cometprotoprivval.Message_PubKeyRequest{PubKeyRequest: pb}
	case *cometprotoprivval.PubKeyResponse:
		msg.Sum = &cometprotoprivval.Message_PubKeyResponse{PubKeyResponse: pb}
	case *cometprotoprivval.SignVoteRequest:
		msg.Sum = &cometprotoprivval.Message_SignVoteRequest{SignVoteRequest: pb}
	case *cometprotoprivval.SignedVoteResponse:
		msg.Sum = &cometprotoprivval.Message_SignedVoteResponse{SignedVoteResponse: pb}
	case *cometprotoprivval.SignedProposalResponse:
		msg.Sum = &cometprotoprivval.Message_SignedProposalResponse{SignedProposalResponse: pb}
	case *cometprotoprivval.SignProposalRequest:
		msg.Sum = &cometprotoprivval.Message_SignProposalRequest{SignProposalRequest: pb}
	case *cometprotoprivval.PingRequest:
		msg.Sum = &cometprotoprivval.Message_PingRequest{PingRequest: pb}
	case *cometprotoprivval.PingResponse:
		msg.Sum = &cometprotoprivval.Message_PingResponse{PingResponse: pb}
	default:
		panic(fmt.Errorf("unknown message type %T", pb))
	}

	return msg
}

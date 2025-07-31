package privval

import (
	"time"

	"github.com/cosmos/gogoproto/proto"
)

// CometBFT v1.0 message types
// These are manually defined to support v1.0 protocol without requiring the full v1.0 dependency

// V1Message is the main message type for v1.0 protocol
type V1Message struct {
	Sum isV1Message_Sum `protobuf_oneof:"sum"`
}

type isV1Message_Sum interface {
	isV1Message_Sum()
}

type V1Message_PubKeyRequest struct {
	PubKeyRequest *V1PubKeyRequest `protobuf:"bytes,1,opt,name=pub_key_request,json=pubKeyRequest,proto3,oneof" json:"pub_key_request,omitempty"`
}

type V1Message_PubKeyResponse struct {
	PubKeyResponse *V1PubKeyResponse `protobuf:"bytes,2,opt,name=pub_key_response,json=pubKeyResponse,proto3,oneof" json:"pub_key_response,omitempty"`
}

type V1Message_SignVoteRequest struct {
	SignVoteRequest *V1SignVoteRequest `protobuf:"bytes,3,opt,name=sign_vote_request,json=signVoteRequest,proto3,oneof" json:"sign_vote_request,omitempty"`
}

type V1Message_SignedVoteResponse struct {
	SignedVoteResponse *V1SignedVoteResponse `protobuf:"bytes,4,opt,name=signed_vote_response,json=signedVoteResponse,proto3,oneof" json:"signed_vote_response,omitempty"`
}

type V1Message_SignProposalRequest struct {
	SignProposalRequest *V1SignProposalRequest `protobuf:"bytes,5,opt,name=sign_proposal_request,json=signProposalRequest,proto3,oneof" json:"sign_proposal_request,omitempty"`
}

type V1Message_SignedProposalResponse struct {
	SignedProposalResponse *V1SignedProposalResponse `protobuf:"bytes,6,opt,name=signed_proposal_response,json=signedProposalResponse,proto3,oneof" json:"signed_proposal_response,omitempty"`
}

type V1Message_PingRequest struct {
	PingRequest *V1PingRequest `protobuf:"bytes,7,opt,name=ping_request,json=pingRequest,proto3,oneof" json:"ping_request,omitempty"`
}

type V1Message_PingResponse struct {
	PingResponse *V1PingResponse `protobuf:"bytes,8,opt,name=ping_response,json=pingResponse,proto3,oneof" json:"ping_response,omitempty"`
}

type V1Message_SignBytesRequest struct {
	SignBytesRequest *V1SignBytesRequest `protobuf:"bytes,1000,opt,name=sign_bytes_request,json=signBytesRequest,proto3,oneof" json:"sign_bytes_request,omitempty"`
}

type V1Message_SignBytesResponse struct {
	SignBytesResponse *V1SignBytesResponse `protobuf:"bytes,1001,opt,name=sign_bytes_response,json=signBytesResponse,proto3,oneof" json:"sign_bytes_response,omitempty"`
}

func (*V1Message_PubKeyRequest) isV1Message_Sum()            {}
func (*V1Message_PubKeyResponse) isV1Message_Sum()           {}
func (*V1Message_SignVoteRequest) isV1Message_Sum()          {}
func (*V1Message_SignedVoteResponse) isV1Message_Sum()       {}
func (*V1Message_SignProposalRequest) isV1Message_Sum()      {}
func (*V1Message_SignedProposalResponse) isV1Message_Sum()   {}
func (*V1Message_PingRequest) isV1Message_Sum()              {}
func (*V1Message_PingResponse) isV1Message_Sum()             {}
func (*V1Message_SignBytesRequest) isV1Message_Sum()         {}
func (*V1Message_SignBytesResponse) isV1Message_Sum()        {}

// V1PubKeyRequest requests the consensus public key from the remote signer.
type V1PubKeyRequest struct {
	ChainId string `protobuf:"bytes,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
}

// V1PubKeyResponse contains the public key for a validator.
type V1PubKeyResponse struct {
	PubKeyBytes []byte              `protobuf:"bytes,1,opt,name=pub_key_bytes,json=pubKeyBytes,proto3" json:"pub_key_bytes,omitempty"`
	PubKeyType  string              `protobuf:"bytes,2,opt,name=pub_key_type,json=pubKeyType,proto3" json:"pub_key_type,omitempty"`
	Error       *V1RemoteSignerError `protobuf:"bytes,3,opt,name=error,proto3" json:"error,omitempty"`
}

// V1SignVoteRequest is a request to sign a vote
type V1SignVoteRequest struct {
	Vote                *V1Vote `protobuf:"bytes,1,opt,name=vote,proto3" json:"vote,omitempty"`
	ChainId             string  `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	SkipExtensionSigning bool    `protobuf:"varint,3,opt,name=skip_extension_signing,json=skipExtensionSigning,proto3" json:"skip_extension_signing,omitempty"`
}

// V1SignedVoteResponse contains a signed vote
type V1SignedVoteResponse struct {
	Vote  *V1Vote              `protobuf:"bytes,1,opt,name=vote,proto3" json:"vote,omitempty"`
	Error *V1RemoteSignerError `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

// V1SignProposalRequest is a request to sign a proposal
type V1SignProposalRequest struct {
	Proposal *V1Proposal `protobuf:"bytes,1,opt,name=proposal,proto3" json:"proposal,omitempty"`
	ChainId  string      `protobuf:"bytes,2,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
}

// V1SignedProposalResponse contains a signed proposal
type V1SignedProposalResponse struct {
	Proposal *V1Proposal          `protobuf:"bytes,1,opt,name=proposal,proto3" json:"proposal,omitempty"`
	Error    *V1RemoteSignerError `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

// V1PingRequest is a request to confirm that the connection is alive.
type V1PingRequest struct{}

// V1PingResponse is a response to confirm that the connection is alive.
type V1PingResponse struct{}

// V1SignBytesRequest is a request to sign arbitrary bytes
type V1SignBytesRequest struct {
	ChainId string `protobuf:"bytes,1,opt,name=chain_id,json=chainId,proto3" json:"chain_id,omitempty"`
	Bytes   []byte `protobuf:"bytes,2,opt,name=bytes,proto3" json:"bytes,omitempty"`
}

// V1SignBytesResponse contains the signature
type V1SignBytesResponse struct {
	Signature []byte              `protobuf:"bytes,1,opt,name=signature,proto3" json:"signature,omitempty"`
	Error     *V1RemoteSignerError `protobuf:"bytes,2,opt,name=error,proto3" json:"error,omitempty"`
}

// V1RemoteSignerError is an error from the remote signer
type V1RemoteSignerError struct {
	Code        int32  `protobuf:"varint,1,opt,name=code,proto3" json:"code,omitempty"`
	Description string `protobuf:"bytes,2,opt,name=description,proto3" json:"description,omitempty"`
}

// V1Vote represents a vote
type V1Vote struct {
	Type                 V1SignedMsgType `protobuf:"varint,1,opt,name=type,proto3,enum=cometbft.types.v1.SignedMsgType" json:"type,omitempty"`
	Height               int64          `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	Round                int32          `protobuf:"varint,3,opt,name=round,proto3" json:"round,omitempty"`
	BlockId              *V1BlockID     `protobuf:"bytes,4,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Timestamp            *time.Time     `protobuf:"bytes,5,opt,name=timestamp,proto3,stdtime" json:"timestamp,omitempty"`
	ValidatorAddress     []byte         `protobuf:"bytes,6,opt,name=validator_address,json=validatorAddress,proto3" json:"validator_address,omitempty"`
	ValidatorIndex       int32          `protobuf:"varint,7,opt,name=validator_index,json=validatorIndex,proto3" json:"validator_index,omitempty"`
	Signature            []byte         `protobuf:"bytes,8,opt,name=signature,proto3" json:"signature,omitempty"`
	Extension            []byte         `protobuf:"bytes,9,opt,name=extension,proto3" json:"extension,omitempty"`
	ExtensionSignature   []byte         `protobuf:"bytes,10,opt,name=extension_signature,json=extensionSignature,proto3" json:"extension_signature,omitempty"`
}

// V1Proposal represents a proposal
type V1Proposal struct {
	Type      V1SignedMsgType `protobuf:"varint,1,opt,name=type,proto3,enum=cometbft.types.v1.SignedMsgType" json:"type,omitempty"`
	Height    int64          `protobuf:"varint,2,opt,name=height,proto3" json:"height,omitempty"`
	Round     int32          `protobuf:"varint,3,opt,name=round,proto3" json:"round,omitempty"`
	PolRound  int32          `protobuf:"varint,4,opt,name=pol_round,json=polRound,proto3" json:"pol_round,omitempty"`
	BlockId   *V1BlockID     `protobuf:"bytes,5,opt,name=block_id,json=blockId,proto3" json:"block_id,omitempty"`
	Timestamp *time.Time     `protobuf:"bytes,6,opt,name=timestamp,proto3,stdtime" json:"timestamp,omitempty"`
	Signature []byte         `protobuf:"bytes,7,opt,name=signature,proto3" json:"signature,omitempty"`
}

// V1BlockID represents a block ID
type V1BlockID struct {
	Hash          []byte          `protobuf:"bytes,1,opt,name=hash,proto3" json:"hash,omitempty"`
	PartSetHeader *V1PartSetHeader `protobuf:"bytes,2,opt,name=part_set_header,json=partSetHeader,proto3" json:"part_set_header,omitempty"`
}

// V1PartSetHeader represents a part set header
type V1PartSetHeader struct {
	Total uint32 `protobuf:"varint,1,opt,name=total,proto3" json:"total,omitempty"`
	Hash  []byte `protobuf:"bytes,2,opt,name=hash,proto3" json:"hash,omitempty"`
}

// V1SignedMsgType is a type of signed message in the consensus.
type V1SignedMsgType int32

const (
	V1SignedMsgTypeUnknown    V1SignedMsgType = 0
	V1SignedMsgTypePrevote    V1SignedMsgType = 1
	V1SignedMsgTypePrecommit  V1SignedMsgType = 2
	V1SignedMsgTypeProposal   V1SignedMsgType = 32
)

// ProtoMessage implementations
func (m *V1Message) ProtoMessage()               {}
func (m *V1Message) Reset()                      { *m = V1Message{} }
func (m *V1Message) String() string              { return proto.CompactTextString(m) }
func (m *V1PubKeyRequest) ProtoMessage()         {}
func (m *V1PubKeyRequest) Reset()                { *m = V1PubKeyRequest{} }
func (m *V1PubKeyRequest) String() string        { return proto.CompactTextString(m) }
func (m *V1PubKeyResponse) ProtoMessage()        {}
func (m *V1PubKeyResponse) Reset()               { *m = V1PubKeyResponse{} }
func (m *V1PubKeyResponse) String() string       { return proto.CompactTextString(m) }
func (m *V1SignVoteRequest) ProtoMessage()       {}
func (m *V1SignVoteRequest) Reset()              { *m = V1SignVoteRequest{} }
func (m *V1SignVoteRequest) String() string      { return proto.CompactTextString(m) }
func (m *V1SignedVoteResponse) ProtoMessage()    {}
func (m *V1SignedVoteResponse) Reset()           { *m = V1SignedVoteResponse{} }
func (m *V1SignedVoteResponse) String() string   { return proto.CompactTextString(m) }
func (m *V1SignProposalRequest) ProtoMessage()   {}
func (m *V1SignProposalRequest) Reset()          { *m = V1SignProposalRequest{} }
func (m *V1SignProposalRequest) String() string  { return proto.CompactTextString(m) }
func (m *V1SignedProposalResponse) ProtoMessage() {}
func (m *V1SignedProposalResponse) Reset()        { *m = V1SignedProposalResponse{} }
func (m *V1SignedProposalResponse) String() string { return proto.CompactTextString(m) }
func (m *V1PingRequest) ProtoMessage()           {}
func (m *V1PingRequest) Reset()                  { *m = V1PingRequest{} }
func (m *V1PingRequest) String() string          { return proto.CompactTextString(m) }
func (m *V1PingResponse) ProtoMessage()          {}
func (m *V1PingResponse) Reset()                 { *m = V1PingResponse{} }
func (m *V1PingResponse) String() string         { return proto.CompactTextString(m) }
func (m *V1RemoteSignerError) ProtoMessage()     {}
func (m *V1RemoteSignerError) Reset()            { *m = V1RemoteSignerError{} }
func (m *V1RemoteSignerError) String() string    { return proto.CompactTextString(m) }
func (m *V1Vote) ProtoMessage()                  {}
func (m *V1Vote) Reset()                         { *m = V1Vote{} }
func (m *V1Vote) String() string                 { return proto.CompactTextString(m) }
func (m *V1Proposal) ProtoMessage()              {}
func (m *V1Proposal) Reset()                     { *m = V1Proposal{} }
func (m *V1Proposal) String() string             { return proto.CompactTextString(m) }
func (m *V1BlockID) ProtoMessage()               {}
func (m *V1BlockID) Reset()                      { *m = V1BlockID{} }
func (m *V1BlockID) String() string              { return proto.CompactTextString(m) }
func (m *V1PartSetHeader) ProtoMessage()         {}
func (m *V1PartSetHeader) Reset()                { *m = V1PartSetHeader{} }
func (m *V1PartSetHeader) String() string        { return proto.CompactTextString(m) }
func (m *V1SignBytesRequest) ProtoMessage()      {}
func (m *V1SignBytesRequest) Reset()             { *m = V1SignBytesRequest{} }
func (m *V1SignBytesRequest) String() string     { return proto.CompactTextString(m) }
func (m *V1SignBytesResponse) ProtoMessage()     {}
func (m *V1SignBytesResponse) Reset()            { *m = V1SignBytesResponse{} }
func (m *V1SignBytesResponse) String() string    { return proto.CompactTextString(m) }
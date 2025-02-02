package pushapi

import (
	"errors"

	"github.com/anyproto/any-sync/net/rpc/rpcerr"
)

var (
	errGroup = rpcerr.ErrGroup(ErrCodes_ErrorOffset)

	ErrUnexpected            = errGroup.Register(errors.New("unexpected error"), uint64(ErrCodes_Unexpected))
	ErrInvalidSignature      = errGroup.Register(errors.New("invalid signature"), uint64(ErrCodes_InvalidSignature))
	ErrInvalidTopicSignature = errGroup.Register(errors.New("invalid topic signature"), uint64(ErrCodes_InvalidTopicSignature))
)

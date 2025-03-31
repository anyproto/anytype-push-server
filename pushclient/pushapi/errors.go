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
	ErrSpaceExists           = errGroup.Register(errors.New("space already exists"), uint64(ErrCodes_SpaceExists))
	ErrNoValidTopics         = errGroup.Register(errors.New("no valid topics"), uint64(ErrCodes_NoValidTopics))
)

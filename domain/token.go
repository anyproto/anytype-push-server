package domain

import "github.com/anyproto/anytype-push-server/pushclient/pushapi"

type Platform uint8

const (
	PlatformIOS     = Platform(pushapi.Platform_IOS)
	PlatformAndroid = Platform(pushapi.Platform_Android)
)

type TokenStatus uint8

const (
	TokenStatusValid TokenStatus = iota
	TokenStatusInvalid
)

type Token struct {
	Id        string      `bson:"_id"`
	AccountId string      `bson:"accountId"`
	PeerId    string      `bson:"peerId"`
	Platform  Platform    `bson:"platform"`
	Status    TokenStatus `bson:"status"`
	Created   int64       `bson:"created"`
	Updated   int64       `bson:"updated"`
}

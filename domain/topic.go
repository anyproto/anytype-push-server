package domain

import (
	"strings"

	"github.com/mr-tron/base58"
)

func NewTopic(spaceKey []byte, topic string) Topic {
	return Topic(base58.Encode(spaceKey) + "/" + topic)
}

type Topic string

func (t Topic) SpaceKeyRaw() ([]byte, error) {
	return base58.Decode(t.SpaceKeyBase58())

}

func (t Topic) SpaceKeyBase58() string {
	if idx := strings.Index(string(t), "/"); idx != -1 {
		return string(t[:idx])
	}
	return ""
}

func (t Topic) Topic() string {
	if idx := strings.Index(string(t), "/"); idx != -1 && idx+1 < len(t) {
		return string(t[idx+1:])
	}
	return ""
}

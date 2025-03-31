package domain

import (
	"strings"

	"github.com/mr-tron/base58"
)

func NewTopic(spaceKey []byte, topic string) Topic {
	return Topic(base58.Encode(spaceKey) + "/" + topic)
}

type Topic string

func (t Topic) SpaceKey() string {
	if idx := strings.Index(string(t), "/"); idx != -1 {
		return string(t[:idx])
	}
	return ""
}

package domain

import (
	"crypto/rand"
	"testing"

	"github.com/mr-tron/base58"
	"github.com/stretchr/testify/assert"
)

func TestTopic_SpaceKey(t *testing.T) {
	spaceKey := make([]byte, 32)
	_, _ = rand.Read(spaceKey)
	topic := NewTopic(spaceKey, "topic")
	res := topic.SpaceKey()
	assert.Equal(t, base58.Encode(spaceKey), res)
}

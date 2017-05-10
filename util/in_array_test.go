package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInArray(t *testing.T) {
	assert.True(t, InArray([]string{"foo", "bar"}, "foo"))
	assert.False(t, InArray([]string{"foo", "bar"}, "foobar"))
}

package util

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInArray(t *testing.T) {
	assert.True(t, Contains([]string{"foo", "bar"}, "foo"))
	assert.False(t, Contains([]string{"foo", "bar"}, "foobar"))
}

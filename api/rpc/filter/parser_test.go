package filter

import (
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestParserFilterString(t *testing.T) {
	p := new(parser)
	_, err := p.parseFilterString("whatever", "hi 'h  ello' >=al> ==d;;'sjfa")
	assert.Nil(t, err)
}

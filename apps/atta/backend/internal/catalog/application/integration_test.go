package application

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewInvoiceSupportPanicsWhenAppMissing(t *testing.T) {
	assert.Panics(t, func() { NewInvoiceSupport(nil) })
}

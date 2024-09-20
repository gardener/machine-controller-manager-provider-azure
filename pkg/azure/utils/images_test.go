package utils

import (
	"testing"

	"github.com/Azure/azure-sdk-for-go/sdk/azcore/to"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	. "github.com/onsi/gomega"
)

func TestToImageReference(t *testing.T) {
	expected := armcompute.ImageReference{
		Publisher: to.Ptr("publisher"),
		Offer:     to.Ptr("offer"),
		SKU:       to.Ptr("sku"),
		Version:   to.Ptr("version"),
	}
	g := NewWithT(t)
	g.Expect(ToImageReference("publisher:offer:sku:version")).To(Equal(expected))
}

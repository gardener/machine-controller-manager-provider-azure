package utils

import "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

// CreateResourceTags changes the tag value to be a pointer to string. Azure APIs require tags to be represented as map[string]*string
func CreateResourceTags(tags map[string]string) map[string]*string {
	vmTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		vmTags[k] = to.Ptr(v)
	}
	return vmTags
}

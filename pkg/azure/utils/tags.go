package utils

import "github.com/Azure/azure-sdk-for-go/sdk/azcore/to"

func CreateResourceTags(tags map[string]string) map[string]*string {
	vmTags := make(map[string]*string, len(tags))
	for k, v := range tags {
		vmTags[k] = to.Ptr(v)
	}
	return vmTags
}

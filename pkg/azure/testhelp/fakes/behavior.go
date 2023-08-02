package fakes

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"k8s.io/klog/v2"
)

type ResourceType string

const (
	VirtualMachinesResourceType   ResourceType = "Microsoft.Compute/virtualMachines"   // as defined in azure
	NetworkInterfacesResourceType ResourceType = "microsoft.network/networkinterfaces" //as defined in azure
	VMImageResourceType           ResourceType = "Microsoft.Compute/VMImage"           // this is not defined in azure, however we have created this to allow defining API behavior for VM Images // this is not defined in azure, however we have created this to allow defining API behavior for VM Images.
	MarketPlaceOrderingOfferType  ResourceType = "Microsoft.MarketplaceOrdering/offertypes"
	SubnetResourceType            ResourceType = "Microsoft.Network/virtualNetworks/subnets"
)

// APIBehaviorSpec allows tests to define custom behavior either for a specific resource or a resource type.
type APIBehaviorSpec struct {
	resourceReactionsByName map[string]map[string]ResourceReaction
	// This is primarily going to be used for resource graph behavior specifications
	// If the query is for a specific type then this map should be populated and used.
	resourceReactionsByType map[ResourceType]map[string]ResourceReaction
}

type ResourceReaction struct {
	timeoutAfter *time.Duration
	panic        bool
	err          error
}

func NewAPIBehaviorSpec() *APIBehaviorSpec {
	return &APIBehaviorSpec{
		resourceReactionsByName: make(map[string]map[string]ResourceReaction),
		resourceReactionsByType: make(map[ResourceType]map[string]ResourceReaction),
	}
}

func (s *APIBehaviorSpec) AddContextTimeoutResourceReaction(resourceName, method string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	return s
}

func (s *APIBehaviorSpec) AddPanicResourceReaction(resourceName, method string) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{panic: true}
	return s
}

func (s *APIBehaviorSpec) AddErrorResourceReaction(resourceName, method string, err error) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{err: err}
	return s
}

func (s *APIBehaviorSpec) SetContextTimeoutReactionsForMethods(resourceName string, methods []string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resourceReactionsByName[resourceName][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	}
	return s
}

func (s *APIBehaviorSpec) SetPanicReactionsForMethods(resourceName string, methods []string) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resourceReactionsByName[resourceName][method] = ResourceReaction{panic: true}
	}
	return s
}

func (s *APIBehaviorSpec) SetErrorReactionsForMethods(resourceName string, methods []string, err error) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resourceReactionsByName[resourceName][method] = ResourceReaction{err: err}
	}
	return s
}

func (s *APIBehaviorSpec) AddContextTimeoutResourceTypeReaction(resourceType ResourceType, method string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	return s
}

func (s *APIBehaviorSpec) AddPanicResourceTypeReaction(resourceType ResourceType, method string) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{panic: true}
	return s
}

func (s *APIBehaviorSpec) AddErrorResourceTypeReaction(resourceType ResourceType, method string, err error) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{err: err}
	return s
}

func (s *APIBehaviorSpec) initializeResourceReactionMapForResource(resourceName string) {
	if _, ok := s.resourceReactionsByName[resourceName]; !ok {
		s.resourceReactionsByName[resourceName] = make(map[string]ResourceReaction)
	}
}

func (s *APIBehaviorSpec) initializeResourceTypeReactionMapForResource(resourceType ResourceType) {
	if _, ok := s.resourceReactionsByType[resourceType]; !ok {
		s.resourceReactionsByType[resourceType] = make(map[string]ResourceReaction)
	}
}

func (s *APIBehaviorSpec) SimulateForResourceType(ctx context.Context, resourceGroup string, resourceType *ResourceType, method string) error {
	resTypeReaction := s.getResourceTypeReaction(resourceType, method)
	return doSimulate(ctx, resTypeReaction, fmt.Sprintf("Panicking for ResourceType -> [resourceGroup: %s, type: %s]", resourceGroup, resourceType))
}

func (s *APIBehaviorSpec) SimulateForResource(ctx context.Context, resourceGroup, resourceName, method string) error {
	resReaction := s.getResourceReaction(resourceName, method)
	return doSimulate(ctx, resReaction, fmt.Sprintf("Panicking for resource -> [resourceGroup: %s, name: %s]", resourceGroup, resourceName))
}

func doSimulate(ctx context.Context, reaction *ResourceReaction, panicMsg string) error {
	if reaction == nil {
		return nil // there is no configured reaction for combination of this method and resourceName
	}
	if reaction.panic {
		panic(panicMsg)
	}
	if reaction.timeoutAfter != nil {
		return testhelp.ContextTimeoutError(ctx, *reaction.timeoutAfter)
	}
	return reaction.err
}

func (s *APIBehaviorSpec) getResourceReaction(resourceName, method string) *ResourceReaction {
	resourceReactionMap, ok := s.resourceReactionsByName[resourceName]
	if !ok {
		return nil
	}
	reaction, ok := resourceReactionMap[method]
	if !ok {
		return nil
	}
	return &reaction
}

func (s *APIBehaviorSpec) getResourceTypeReaction(resourceType *ResourceType, method string) *ResourceReaction {
	// This will result in a search across all resource types, first reaction matching method will be returned
	if resourceType == nil {
		klog.Infof("(getResourceTypeReaction) resourceType passed is nil, will return the first set reaction for the method: %s", method)
		for _, reactionsMap := range s.resourceReactionsByType {
			reaction, ok := reactionsMap[method]
			if ok {
				return &reaction
			}
		}
		return nil
	}
	resourceTypeReactionMap, ok := s.resourceReactionsByType[*resourceType]
	if !ok {
		return nil
	}
	reaction, ok := resourceTypeReactionMap[method]
	if !ok {
		return nil
	}
	return &reaction
}

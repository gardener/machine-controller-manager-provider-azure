// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package fakes

import (
	"context"
	"fmt"
	"time"

	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/testhelp"
	"github.com/gardener/machine-controller-manager-provider-azure/pkg/azure/utils"
	"k8s.io/klog/v2"
)

// APIBehaviorSpec allows tests to define custom behavior either for a specific resource or a resource type.
type APIBehaviorSpec struct {
	resourceReactionsByName map[string]map[string]ResourceReaction
	// This is primarily going to be used for resource graph behavior specifications
	// If the query is for a specific type then this map should be populated and used.
	resourceReactionsByType map[utils.ResourceType]map[string]ResourceReaction
}

// ResourceReaction captures reaction for a resource.
// Consumers can define a panic or a context timeout or an error for a specific resource.
type ResourceReaction struct {
	timeoutAfter *time.Duration
	panic        bool
	err          error
}

// NewAPIBehaviorSpec creates a new APIBehaviorSpec.
func NewAPIBehaviorSpec() *APIBehaviorSpec {
	return &APIBehaviorSpec{
		resourceReactionsByName: make(map[string]map[string]ResourceReaction),
		resourceReactionsByType: make(map[utils.ResourceType]map[string]ResourceReaction),
	}
}

// AddContextTimeoutResourceReaction adds a context timeout reaction for a resource when the given method is invoked on the respective resource client.
// The timeout should happen after the timeout duration passed to this method.
func (s *APIBehaviorSpec) AddContextTimeoutResourceReaction(resourceName, method string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	return s
}

// AddPanicResourceReaction adds a panic reaction for a resource when a given method is invoked on the respective resource client.
func (s *APIBehaviorSpec) AddPanicResourceReaction(resourceName, method string) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{panic: true}
	return s
}

// AddErrorResourceReaction adds an error reaction for a resource returning the error passed as an argument when the given method is invoked on the respective resource client.
func (s *APIBehaviorSpec) AddErrorResourceReaction(resourceName, method string, err error) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resourceReactionsByName[resourceName][method] = ResourceReaction{err: err}
	return s
}

// AddContextTimeoutResourceTypeReaction adds a context timeout reaction for all resources of the given resourceType.
// Context timeout is simulated after the given timeoutAfter duration when the given method on the resource client is invoked.
func (s *APIBehaviorSpec) AddContextTimeoutResourceTypeReaction(resourceType utils.ResourceType, method string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	return s
}

// AddPanicResourceTypeReaction adds a panic reaction for all resources of a given resourceType when a given method on the resource client is invoked.
func (s *APIBehaviorSpec) AddPanicResourceTypeReaction(resourceType utils.ResourceType, method string) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{panic: true}
	return s
}

// AddErrorResourceTypeReaction adds an error reaction for all resources of a given resourceType. The give error is returned
// when the given method is invoked on the respective resource client.
func (s *APIBehaviorSpec) AddErrorResourceTypeReaction(resourceType utils.ResourceType, method string, err error) *APIBehaviorSpec {
	s.initializeResourceTypeReactionMapForResource(resourceType)
	s.resourceReactionsByType[resourceType][method] = ResourceReaction{err: err}
	return s
}

func (s *APIBehaviorSpec) initializeResourceReactionMapForResource(resourceName string) {
	if _, ok := s.resourceReactionsByName[resourceName]; !ok {
		s.resourceReactionsByName[resourceName] = make(map[string]ResourceReaction)
	}
}

func (s *APIBehaviorSpec) initializeResourceTypeReactionMapForResource(resourceType utils.ResourceType) {
	if _, ok := s.resourceReactionsByType[resourceType]; !ok {
		s.resourceReactionsByType[resourceType] = make(map[string]ResourceReaction)
	}
}

// SimulateForResourceType runs the simulation for a resourceType and method combination using any configured reactions.
func (s *APIBehaviorSpec) SimulateForResourceType(ctx context.Context, resourceGroup string, resourceType *utils.ResourceType, method string) error {
	resTypeReaction := s.getResourceTypeReaction(resourceType, method)
	return doSimulate(ctx, resTypeReaction, fmt.Sprintf("Panicking for ResourceType -> [resourceGroup: %s, type: %s]", resourceGroup, *resourceType))
}

// SimulateForResource runs the simulation for a resource and method combination using any configured reactions.
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

func (s *APIBehaviorSpec) getResourceTypeReaction(resourceType *utils.ResourceType, method string) *ResourceReaction {
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

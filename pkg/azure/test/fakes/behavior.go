package fakes

import (
	"context"
	"fmt"
	"time"
)

type APIBehaviorSpec struct {
	resources map[string]map[string]ResourceReaction
}

type ResourceReaction struct {
	timeoutAfter *time.Duration
	panic        bool
	err          error
}

func NewAPIBehaviorSpec() *APIBehaviorSpec {
	return &APIBehaviorSpec{resources: make(map[string]map[string]ResourceReaction)}
}

func (s *APIBehaviorSpec) AddContextTimeoutResourceReaction(resourceName, method string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resources[resourceName][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	return s
}

func (s *APIBehaviorSpec) AddPanicResourceReaction(resourceName, method string) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resources[resourceName][method] = ResourceReaction{panic: true}
	return s
}

func (s *APIBehaviorSpec) AddErrorResourceReaction(resourceName, method string, err error) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	s.resources[resourceName][method] = ResourceReaction{err: err}
	return s
}

func (s *APIBehaviorSpec) SetContextTimeoutReactionsForMethods(resourceName string, methods []string, timeoutAfter time.Duration) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resources[resourceName][method] = ResourceReaction{timeoutAfter: &timeoutAfter}
	}
	return s
}

func (s *APIBehaviorSpec) SetPanicReactionsForMethods(resourceName string, methods []string) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resources[resourceName][method] = ResourceReaction{panic: true}
	}
	return s
}

func (s *APIBehaviorSpec) SetErrorReactionsForMethods(resourceName string, methods []string, err error) *APIBehaviorSpec {
	s.initializeResourceReactionMapForResource(resourceName)
	for _, method := range methods {
		s.resources[resourceName][method] = ResourceReaction{err: err}
	}
	return s
}

func (s *APIBehaviorSpec) initializeResourceReactionMapForResource(resourceName string) {
	if _, ok := s.resources[resourceName]; !ok {
		s.resources[resourceName] = make(map[string]ResourceReaction)
	}
}

func (s *APIBehaviorSpec) Simulate(ctx context.Context, resourceGroup, resourceName, method string) error {
	resReaction := s.getResourceReaction(resourceName, method)
	if resReaction.panic {
		panic(fmt.Sprintf("Panicking for resource -> [resourceGroup: %s, name: %s]", resourceGroup, resourceName))
	}
	if resReaction.timeoutAfter != nil {
		return ContextTimeoutError(ctx, *resReaction.timeoutAfter)
	}
	return resReaction.err
}

func (s *APIBehaviorSpec) getResourceReaction(resourceName, method string) *ResourceReaction {
	resourceReactionMap, ok := s.resources[resourceName]
	if !ok {
		return nil
	}
	reaction, ok := resourceReactionMap[method]
	if !ok {
		return nil
	}
	return &reaction
}

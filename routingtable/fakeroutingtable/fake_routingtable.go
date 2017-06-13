// This file was generated by counterfeiter
package fakeroutingtable

import (
	"sync"

	modelsbbs "code.cloudfoundry.org/bbs/models"
	"code.cloudfoundry.org/route-emitter/routingtable"
	"code.cloudfoundry.org/route-emitter/routingtable/schema/endpoint"
)

type FakeRoutingTable struct {
	AddEndpointStub        func(actualLRP *endpoint.ActualLRPRoutingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	addEndpointMutex       sync.RWMutex
	addEndpointArgsForCall []struct {
		actualLRP *endpoint.ActualLRPRoutingInfo
	}
	addEndpointReturns struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	RemoveEndpointStub        func(actualLRP *endpoint.ActualLRPRoutingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	removeEndpointMutex       sync.RWMutex
	removeEndpointArgsForCall []struct {
		actualLRP *endpoint.ActualLRPRoutingInfo
	}
	removeEndpointReturns struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	SwapStub        func(t routingtable.RoutingTable, domains modelsbbs.DomainSet) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	swapMutex       sync.RWMutex
	swapArgsForCall []struct {
		t       routingtable.RoutingTable
		domains modelsbbs.DomainSet
	}
	swapReturns struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	EmitStub        func() (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	emitMutex       sync.RWMutex
	emitArgsForCall []struct{}
	emitReturns     struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	EndpointsForIndexStub        func(key endpoint.RoutingKey, index int32) []routingtable.Endpoint
	endpointsForIndexMutex       sync.RWMutex
	endpointsForIndexArgsForCall []struct {
		key   endpoint.RoutingKey
		index int32
	}
	endpointsForIndexReturns struct {
		result1 []routingtable.Endpoint
	}
	SetRoutesStub        func(beforeLRP, afterLRP *modelsbbs.DesiredLRPSchedulingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	setRoutesMutex       sync.RWMutex
	setRoutesArgsForCall []struct {
		beforeLRP *modelsbbs.DesiredLRPSchedulingInfo
		afterLRP  *modelsbbs.DesiredLRPSchedulingInfo
	}
	setRoutesReturns struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	RemoveRoutesStub        func(desiredLRP *modelsbbs.DesiredLRPSchedulingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit)
	removeRoutesMutex       sync.RWMutex
	removeRoutesArgsForCall []struct {
		desiredLRP *modelsbbs.DesiredLRPSchedulingInfo
	}
	removeRoutesReturns struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}
	HTTPEndpointCountStub        func() int
	hTTPEndpointCountMutex       sync.RWMutex
	hTTPEndpointCountArgsForCall []struct{}
	hTTPEndpointCountReturns     struct {
		result1 int
	}
	TCPRouteCountStub        func() int
	tCPRouteCountMutex       sync.RWMutex
	tCPRouteCountArgsForCall []struct{}
	tCPRouteCountReturns     struct {
		result1 int
	}
	invocations      map[string][][]interface{}
	invocationsMutex sync.RWMutex
}

func (fake *FakeRoutingTable) AddEndpoint(actualLRP *endpoint.ActualLRPRoutingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.addEndpointMutex.Lock()
	fake.addEndpointArgsForCall = append(fake.addEndpointArgsForCall, struct {
		actualLRP *endpoint.ActualLRPRoutingInfo
	}{actualLRP})
	fake.recordInvocation("AddEndpoint", []interface{}{actualLRP})
	fake.addEndpointMutex.Unlock()
	if fake.AddEndpointStub != nil {
		return fake.AddEndpointStub(actualLRP)
	} else {
		return fake.addEndpointReturns.result1, fake.addEndpointReturns.result2
	}
}

func (fake *FakeRoutingTable) AddEndpointCallCount() int {
	fake.addEndpointMutex.RLock()
	defer fake.addEndpointMutex.RUnlock()
	return len(fake.addEndpointArgsForCall)
}

func (fake *FakeRoutingTable) AddEndpointArgsForCall(i int) *endpoint.ActualLRPRoutingInfo {
	fake.addEndpointMutex.RLock()
	defer fake.addEndpointMutex.RUnlock()
	return fake.addEndpointArgsForCall[i].actualLRP
}

func (fake *FakeRoutingTable) AddEndpointReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.AddEndpointStub = nil
	fake.addEndpointReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) RemoveEndpoint(actualLRP *endpoint.ActualLRPRoutingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.removeEndpointMutex.Lock()
	fake.removeEndpointArgsForCall = append(fake.removeEndpointArgsForCall, struct {
		actualLRP *endpoint.ActualLRPRoutingInfo
	}{actualLRP})
	fake.recordInvocation("RemoveEndpoint", []interface{}{actualLRP})
	fake.removeEndpointMutex.Unlock()
	if fake.RemoveEndpointStub != nil {
		return fake.RemoveEndpointStub(actualLRP)
	} else {
		return fake.removeEndpointReturns.result1, fake.removeEndpointReturns.result2
	}
}

func (fake *FakeRoutingTable) RemoveEndpointCallCount() int {
	fake.removeEndpointMutex.RLock()
	defer fake.removeEndpointMutex.RUnlock()
	return len(fake.removeEndpointArgsForCall)
}

func (fake *FakeRoutingTable) RemoveEndpointArgsForCall(i int) *endpoint.ActualLRPRoutingInfo {
	fake.removeEndpointMutex.RLock()
	defer fake.removeEndpointMutex.RUnlock()
	return fake.removeEndpointArgsForCall[i].actualLRP
}

func (fake *FakeRoutingTable) RemoveEndpointReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.RemoveEndpointStub = nil
	fake.removeEndpointReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) Swap(t routingtable.RoutingTable, domains modelsbbs.DomainSet) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.swapMutex.Lock()
	fake.swapArgsForCall = append(fake.swapArgsForCall, struct {
		t       routingtable.RoutingTable
		domains modelsbbs.DomainSet
	}{t, domains})
	fake.recordInvocation("Swap", []interface{}{t, domains})
	fake.swapMutex.Unlock()
	if fake.SwapStub != nil {
		return fake.SwapStub(t, domains)
	} else {
		return fake.swapReturns.result1, fake.swapReturns.result2
	}
}

func (fake *FakeRoutingTable) SwapCallCount() int {
	fake.swapMutex.RLock()
	defer fake.swapMutex.RUnlock()
	return len(fake.swapArgsForCall)
}

func (fake *FakeRoutingTable) SwapArgsForCall(i int) (routingtable.RoutingTable, modelsbbs.DomainSet) {
	fake.swapMutex.RLock()
	defer fake.swapMutex.RUnlock()
	return fake.swapArgsForCall[i].t, fake.swapArgsForCall[i].domains
}

func (fake *FakeRoutingTable) SwapReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.SwapStub = nil
	fake.swapReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) Emit() (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.emitMutex.Lock()
	fake.emitArgsForCall = append(fake.emitArgsForCall, struct{}{})
	fake.recordInvocation("Emit", []interface{}{})
	fake.emitMutex.Unlock()
	if fake.EmitStub != nil {
		return fake.EmitStub()
	} else {
		return fake.emitReturns.result1, fake.emitReturns.result2
	}
}

func (fake *FakeRoutingTable) EmitCallCount() int {
	fake.emitMutex.RLock()
	defer fake.emitMutex.RUnlock()
	return len(fake.emitArgsForCall)
}

func (fake *FakeRoutingTable) EmitReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.EmitStub = nil
	fake.emitReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) EndpointsForIndex(key endpoint.RoutingKey, index int32) []routingtable.Endpoint {
	fake.endpointsForIndexMutex.Lock()
	fake.endpointsForIndexArgsForCall = append(fake.endpointsForIndexArgsForCall, struct {
		key   endpoint.RoutingKey
		index int32
	}{key, index})
	fake.recordInvocation("EndpointsForIndex", []interface{}{key, index})
	fake.endpointsForIndexMutex.Unlock()
	if fake.EndpointsForIndexStub != nil {
		return fake.EndpointsForIndexStub(key, index)
	} else {
		return fake.endpointsForIndexReturns.result1
	}
}

func (fake *FakeRoutingTable) EndpointsForIndexCallCount() int {
	fake.endpointsForIndexMutex.RLock()
	defer fake.endpointsForIndexMutex.RUnlock()
	return len(fake.endpointsForIndexArgsForCall)
}

func (fake *FakeRoutingTable) EndpointsForIndexArgsForCall(i int) (endpoint.RoutingKey, int32) {
	fake.endpointsForIndexMutex.RLock()
	defer fake.endpointsForIndexMutex.RUnlock()
	return fake.endpointsForIndexArgsForCall[i].key, fake.endpointsForIndexArgsForCall[i].index
}

func (fake *FakeRoutingTable) EndpointsForIndexReturns(result1 []routingtable.Endpoint) {
	fake.EndpointsForIndexStub = nil
	fake.endpointsForIndexReturns = struct {
		result1 []routingtable.Endpoint
	}{result1}
}

func (fake *FakeRoutingTable) SetRoutes(beforeLRP *modelsbbs.DesiredLRPSchedulingInfo, afterLRP *modelsbbs.DesiredLRPSchedulingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.setRoutesMutex.Lock()
	fake.setRoutesArgsForCall = append(fake.setRoutesArgsForCall, struct {
		beforeLRP *modelsbbs.DesiredLRPSchedulingInfo
		afterLRP  *modelsbbs.DesiredLRPSchedulingInfo
	}{beforeLRP, afterLRP})
	fake.recordInvocation("SetRoutes", []interface{}{beforeLRP, afterLRP})
	fake.setRoutesMutex.Unlock()
	if fake.SetRoutesStub != nil {
		return fake.SetRoutesStub(beforeLRP, afterLRP)
	} else {
		return fake.setRoutesReturns.result1, fake.setRoutesReturns.result2
	}
}

func (fake *FakeRoutingTable) SetRoutesCallCount() int {
	fake.setRoutesMutex.RLock()
	defer fake.setRoutesMutex.RUnlock()
	return len(fake.setRoutesArgsForCall)
}

func (fake *FakeRoutingTable) SetRoutesArgsForCall(i int) (*modelsbbs.DesiredLRPSchedulingInfo, *modelsbbs.DesiredLRPSchedulingInfo) {
	fake.setRoutesMutex.RLock()
	defer fake.setRoutesMutex.RUnlock()
	return fake.setRoutesArgsForCall[i].beforeLRP, fake.setRoutesArgsForCall[i].afterLRP
}

func (fake *FakeRoutingTable) SetRoutesReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.SetRoutesStub = nil
	fake.setRoutesReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) RemoveRoutes(desiredLRP *modelsbbs.DesiredLRPSchedulingInfo) (routingtable.TCPRouteMappings, routingtable.MessagesToEmit) {
	fake.removeRoutesMutex.Lock()
	fake.removeRoutesArgsForCall = append(fake.removeRoutesArgsForCall, struct {
		desiredLRP *modelsbbs.DesiredLRPSchedulingInfo
	}{desiredLRP})
	fake.recordInvocation("RemoveRoutes", []interface{}{desiredLRP})
	fake.removeRoutesMutex.Unlock()
	if fake.RemoveRoutesStub != nil {
		return fake.RemoveRoutesStub(desiredLRP)
	} else {
		return fake.removeRoutesReturns.result1, fake.removeRoutesReturns.result2
	}
}

func (fake *FakeRoutingTable) RemoveRoutesCallCount() int {
	fake.removeRoutesMutex.RLock()
	defer fake.removeRoutesMutex.RUnlock()
	return len(fake.removeRoutesArgsForCall)
}

func (fake *FakeRoutingTable) RemoveRoutesArgsForCall(i int) *modelsbbs.DesiredLRPSchedulingInfo {
	fake.removeRoutesMutex.RLock()
	defer fake.removeRoutesMutex.RUnlock()
	return fake.removeRoutesArgsForCall[i].desiredLRP
}

func (fake *FakeRoutingTable) RemoveRoutesReturns(result1 routingtable.TCPRouteMappings, result2 routingtable.MessagesToEmit) {
	fake.RemoveRoutesStub = nil
	fake.removeRoutesReturns = struct {
		result1 routingtable.TCPRouteMappings
		result2 routingtable.MessagesToEmit
	}{result1, result2}
}

func (fake *FakeRoutingTable) HTTPEndpointCount() int {
	fake.hTTPEndpointCountMutex.Lock()
	fake.hTTPEndpointCountArgsForCall = append(fake.hTTPEndpointCountArgsForCall, struct{}{})
	fake.recordInvocation("HTTPEndpointCount", []interface{}{})
	fake.hTTPEndpointCountMutex.Unlock()
	if fake.HTTPEndpointCountStub != nil {
		return fake.HTTPEndpointCountStub()
	} else {
		return fake.hTTPEndpointCountReturns.result1
	}
}

func (fake *FakeRoutingTable) HTTPEndpointCountCallCount() int {
	fake.hTTPEndpointCountMutex.RLock()
	defer fake.hTTPEndpointCountMutex.RUnlock()
	return len(fake.hTTPEndpointCountArgsForCall)
}

func (fake *FakeRoutingTable) HTTPEndpointCountReturns(result1 int) {
	fake.HTTPEndpointCountStub = nil
	fake.hTTPEndpointCountReturns = struct {
		result1 int
	}{result1}
}

func (fake *FakeRoutingTable) TCPRouteCount() int {
	fake.tCPRouteCountMutex.Lock()
	fake.tCPRouteCountArgsForCall = append(fake.tCPRouteCountArgsForCall, struct{}{})
	fake.recordInvocation("TCPRouteCount", []interface{}{})
	fake.tCPRouteCountMutex.Unlock()
	if fake.TCPRouteCountStub != nil {
		return fake.TCPRouteCountStub()
	} else {
		return fake.tCPRouteCountReturns.result1
	}
}

func (fake *FakeRoutingTable) TCPRouteCountCallCount() int {
	fake.tCPRouteCountMutex.RLock()
	defer fake.tCPRouteCountMutex.RUnlock()
	return len(fake.tCPRouteCountArgsForCall)
}

func (fake *FakeRoutingTable) TCPRouteCountReturns(result1 int) {
	fake.TCPRouteCountStub = nil
	fake.tCPRouteCountReturns = struct {
		result1 int
	}{result1}
}

func (fake *FakeRoutingTable) Invocations() map[string][][]interface{} {
	fake.invocationsMutex.RLock()
	defer fake.invocationsMutex.RUnlock()
	fake.addEndpointMutex.RLock()
	defer fake.addEndpointMutex.RUnlock()
	fake.removeEndpointMutex.RLock()
	defer fake.removeEndpointMutex.RUnlock()
	fake.swapMutex.RLock()
	defer fake.swapMutex.RUnlock()
	fake.emitMutex.RLock()
	defer fake.emitMutex.RUnlock()
	fake.endpointsForIndexMutex.RLock()
	defer fake.endpointsForIndexMutex.RUnlock()
	fake.setRoutesMutex.RLock()
	defer fake.setRoutesMutex.RUnlock()
	fake.removeRoutesMutex.RLock()
	defer fake.removeRoutesMutex.RUnlock()
	fake.hTTPEndpointCountMutex.RLock()
	defer fake.hTTPEndpointCountMutex.RUnlock()
	fake.tCPRouteCountMutex.RLock()
	defer fake.tCPRouteCountMutex.RUnlock()
	return fake.invocations
}

func (fake *FakeRoutingTable) recordInvocation(key string, args []interface{}) {
	fake.invocationsMutex.Lock()
	defer fake.invocationsMutex.Unlock()
	if fake.invocations == nil {
		fake.invocations = map[string][][]interface{}{}
	}
	if fake.invocations[key] == nil {
		fake.invocations[key] = [][]interface{}{}
	}
	fake.invocations[key] = append(fake.invocations[key], args)
}

var _ routingtable.RoutingTable = new(FakeRoutingTable)
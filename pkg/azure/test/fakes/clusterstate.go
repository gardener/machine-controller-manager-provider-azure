package fakes

import (
	"sync"
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
)

type APIBehaviorOptions struct {
	TimeoutAfter *time.Duration
}

type ClusterState struct {
	mutex               sync.RWMutex
	ResourceGroup       string
	MachineResourcesMap map[string]MachineResources
}

type DiskType string

const (
	DiskTypeOS   DiskType = "OSDisk"
	DiskTypeData DiskType = "DataDisk"
)

func NewClusterState(resourceGroup string) *ClusterState {
	return &ClusterState{
		ResourceGroup:       resourceGroup,
		MachineResourcesMap: make(map[string]MachineResources),
	}
}

func (c *ClusterState) AddMachineResources(m MachineResources) {
	c.MachineResourcesMap[m.Name] = m
}

func (c *ClusterState) GetVM(vmName string) *armcompute.VirtualMachine {
	if machineResources, ok := c.MachineResourcesMap[vmName]; ok {
		return machineResources.VM
	}
	return nil
}

func (c *ClusterState) DeleteVM(vmName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	m, ok := c.MachineResourcesMap[vmName]
	if !ok {
		return
	}
	if m.ShouldCascadeDeleteAllAttachedResources() {
		delete(c.MachineResourcesMap, vmName)
		return
	}

	m.VM = nil
	m.HandleNICOnMachineDelete()
	m.HandleOSDiskOnMachineDelete()
	m.HandleDataDisksOnMachineDelete()

	if !m.HasResources() {
		delete(c.MachineResourcesMap, vmName)
	} else {
		c.MachineResourcesMap[vmName] = m
	}
}

func (c *ClusterState) GetNIC(nicName string) *armnetwork.Interface {
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			return m.NIC
		}
	}
	return nil
}

func (c *ClusterState) DeleteNIC(nicName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	var targetMachineResources *MachineResources
loop:
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			targetMachineResources = &m
			break loop
		}
	}
	if targetMachineResources != nil {
		targetMachineResources.NIC = nil
		if !targetMachineResources.HasResources() {
			delete(c.MachineResourcesMap, targetMachineResources.Name)
		}
		c.MachineResourcesMap[targetMachineResources.Name] = *targetMachineResources
	}
}

func (c *ClusterState) GetDisk(diskName string) *armcompute.Disk {
	diskType, machine := c.getDiskTypeAndOwningMachineResources(diskName)
	switch diskType {
	case DiskTypeOS:
		return machine.OSDisk
	case DiskTypeData:
		return machine.DataDisks[diskName]
	default:
		return nil
	}
}

func (c *ClusterState) DeleteDisk(diskName string) {
	c.mutex.Lock()
	defer c.mutex.Unlock()
	diskType, machineResources := c.getDiskTypeAndOwningMachineResources(diskName)
	if machineResources == nil {
		return
	}
	switch diskType {
	case DiskTypeOS:
		machineResources.OSDisk = nil
	case DiskTypeData:
		delete(machineResources.DataDisks, diskName)
		if len(machineResources.DataDisks) == 0 {
			machineResources.DataDisks = nil
		}
	}
	if !machineResources.HasResources() {
		delete(c.MachineResourcesMap, machineResources.Name)
	} else {
		c.MachineResourcesMap[machineResources.Name] = *machineResources
	}
}

func (c *ClusterState) getDiskTypeAndOwningMachineResources(diskName string) (DiskType, *MachineResources) {
	if c.MachineResourcesMap != nil {
		for _, m := range c.MachineResourcesMap {
			if m.OSDisk != nil && *m.OSDisk.Name == diskName {
				return DiskTypeOS, &m
			}
			if m.DataDisks != nil {
				if _, ok := m.DataDisks[diskName]; ok {
					return DiskTypeData, &m
				}
			}
		}
	}
	return "", nil
}

package fakes

import (
	"time"

	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/compute/armcompute/v5"
	"github.com/Azure/azure-sdk-for-go/sdk/resourcemanager/network/armnetwork/v3"
)

type APIBehaviorOptions struct {
	TimeoutAfter *time.Duration
}

type ClusterState struct {
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
	c.MachineResourcesMap[*m.VM.Name] = m
}

func (c *ClusterState) DeleteVM(vmName string) {
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

	if m.HasResources() {
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
	var targetMachine *MachineResources
loop:
	for _, m := range c.MachineResourcesMap {
		if m.NIC != nil && *m.NIC.Name == nicName {
			targetMachine = &m
			break loop
		}
	}
	if targetMachine != nil {
		targetMachine.NIC = nil
		if !targetMachine.HasResources() {
			delete(c.MachineResourcesMap, *targetMachine.VM.Name)
		}
	}
}

func (c *ClusterState) GetDisk(diskName string) *armcompute.Disk {
	diskType, machine := c.getDiskTypeAndOwningMachine(diskName)
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
	diskType, machine := c.getDiskTypeAndOwningMachine(diskName)
	if machine == nil {
		return
	}
	switch diskType {
	case DiskTypeOS:
		machine.OSDisk = nil
	case DiskTypeData:
		delete(machine.DataDisks, diskName)
		if len(machine.DataDisks) == 0 {
			machine.DataDisks = nil
		}
	}
	if !machine.HasResources() {
		delete(c.MachineResourcesMap, *machine.VM.Name)
	} else {
		c.MachineResourcesMap[machine.Name] = *machine
	}
}

func (c *ClusterState) getDiskTypeAndOwningMachine(diskName string) (DiskType, *MachineResources) {
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

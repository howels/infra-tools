package vsphere

import (
	"context"
	"fmt"
	"log"
	"strconv"

	//. "github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

//VMPassthrough adds a Host PCI device to a VM.
func VMPassthrough(ctx context.Context, host *object.HostSystem, vm *object.VirtualMachine, client *vim25.Client) error {
	pc := property.DefaultCollector(client)
	var envbrowsermoref types.ManagedObjectReference
	err := vm.Properties(ctx, vm.Reference(), []string{"environmentBrowser"}, &envbrowsermoref)
	if err != nil {
		log.Print("Failed to retrieve ESXi hostsystem object")
		return err
	}
	var envBrowsers []*mo.EnvironmentBrowser
	err = pc.Retrieve(ctx, []types.ManagedObjectReference{envbrowsermoref}, nil, &envBrowsers)
	if err != nil {
		return err
	}
	if len(envBrowsers) != 1 {
		return fmt.Errorf("Failed to obtain environment browser for this VM")
	}
	envBrowser := envBrowsers[0]
	hostRef := host.Reference()
	req := types.QueryConfigTarget{
		This: envBrowser.Reference(),
		Host: &hostRef,
	}
	res, err := methods.QueryConfigTarget(ctx, client.RoundTripper, &req)
	if err != nil {
		return err
	}
	devs := res.Returnval.PciPassthrough
	if len(devs) != 1 {
		return fmt.Errorf("Found >1 or 0 available PCI passthrough devices when only 1 is expected")
	}
	dev := devs[0].GetVirtualMachinePciPassthroughInfo()

	description := types.Description{
		Label:   "PCI device 0",
		Summary: "",
	}

	backing := types.VirtualPCIPassthroughDeviceBackingInfo{
		Id:       dev.PciDevice.Id,
		DeviceId: strconv.FormatInt(int64(dev.PciDevice.DeviceId), 16), // base16 or hex version of the deviceId
		VendorId: dev.PciDevice.VendorId,
		SystemId: dev.SystemId, // god knows what this is but it works.
	}

	vmPciDevice := types.VirtualPCIPassthrough{
		VirtualDevice: types.VirtualDevice{
			Key:        -100,
			DeviceInfo: description.GetDescription(),
			Backing:    backing.GetVirtualDeviceDeviceBackingInfo(),
		},
	}

	devChange := types.VirtualDeviceConfigSpec{Device: vmPciDevice.GetVirtualDevice(), Operation: "add"}

	vmConfigSpec := types.VirtualMachineConfigSpec{
		DeviceChange: []types.BaseVirtualDeviceConfigSpec{devChange.GetVirtualDeviceConfigSpec()},
	}

	task, err := vm.Reconfigure(ctx, vmConfigSpec)
	if err != nil {
		return fmt.Errorf("Could not send config spec to VM")
	}
	if _, err = task.WaitForResult(ctx, nil); err != nil {
		log.Print("Failed to configure VM: " + vm.Name())
		return err
	}
	return nil
}

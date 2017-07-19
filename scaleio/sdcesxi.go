package scaleio

import (
	"context"
	"log"
	"reflect"
	"strings"

	"github.com/vmware/govmomi/examples"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/view"
	// "github.com/vmware/govmomi"
	// "github.com/vmware/govmomi/find"
	// "github.com/vmware/govmomi/object"
	// "github.com/vmware/govmomi/ovf"
	// "github.com/vmware/govmomi/vim25"
	// "github.com/vmware/govmomi/vim25/progress"
	// "github.com/vmware/govmomi/vim25/soap"
	// "github.com/vmware/govmomi/vim25/types"
	"fmt"

	"github.com/howels/infra-tools/ssh"
	"github.com/howels/infra-tools/vsphere"
	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/vim25/methods"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

//SDCESXi describes the ESXi host being used as ScaleIO init or SDC.
type SDCESXi struct {
	HostSystem  *object.HostSystem
	Network     NetworkMap
	MdmIPString string
	IniGUIDStr  string
	Hostname    string
	SSH         sshclient.ShellConnection
	Vcenter     *vsphere.Vcenter
}

//Command allows for SSH commands to be sent to the ESXI server
func (sdc *SDCESXi) Command(cmd string) (*sshclient.CommandOutput, error) {
	// s := sshclient.NewSSHClient("sshtest", "sshtest", "localhost")
	out, err := sdc.SSH.Command(cmd)
	if err != nil {
		return out, err
	}
	fmt.Println(string(out.Stdout))
	return out, nil
}

//UpdateScini writes values to the scini module configuration in ESXi
func (sdc *SDCESXi) UpdateScini(mdmIP string, guid string) error {
	if sdc.MdmIPString == "" {
		return fmt.Errorf("No MDM IP assigned for SDCESXi, cannot set guid without MDMIPString")
	}
	_, err := sdc.Command(fmt.Sprintf("esxcli system module parameters set -m scini -p 'IoctlIniGuidStr=%v IoctlMdmIPStr=%v'", mdmIP, guid))
	if err != nil {
		return err
	}
	sdc.IniGUIDStr = guid
	sdc.MdmIPString = mdmIP
	_, err = sdc.Command("vmkload_mod -u scini;esxcli system module load -m scini")
	if err != nil {
		return err
	}

	return nil

}

//EnablePassthrough makes a hardware device available to VMs if it matches a name pattern
func (sdc *SDCESXi) EnablePassthrough(devname string) error {
	err := sdc.Vcenter.Login()
	if err != nil {
		log.Print("vCenter login failed")
		return err
	}
	//defer sdc.Vcenter.Logout()
	client := sdc.Vcenter.Client
	ctx := sdc.Vcenter.Context

	var h mo.HostSystem

	//no govmomi support for the pcpassthrusystem objects so gotta get the underlying managed objects
	err = sdc.HostSystem.Properties(ctx, sdc.HostSystem.Reference(), []string{"configManager.pciPassthruSystem", "hardware.pciDevice"}, &h)
	if err != nil {
		log.Print("Failed to retrieve ESXi hostsystem object")
		return err
	}
	passthrough := h.ConfigManager.PciPassthruSystem
	log.Printf("Passthrough data: %+v", passthrough.Reference())
	list := []types.ManagedObjectReference{*passthrough}

	// log.Printf("View: %+v", v)
	pc := property.DefaultCollector(client.Client)
	var p []mo.HostPciPassthruSystem
	err = pc.Retrieve(ctx, list, nil, &p)
	if err != nil {
		return err
	}
	//log.Printf("Object: %+v", p)
	passthroughSystem := p[0]
	for _, dev := range passthroughSystem.PciPassthruInfo {
		info := dev.GetHostPciPassthruInfo()
		if info.PassthruEnabled {
			log.Printf("PCI pass through already enabled: %+v", info)
		}
		if info.PassthruActive {
			log.Printf("PCI pass through already active: %+v", info)
			return nil
		}
	}

	var targetDev types.HostPciDevice
	for _, dev := range h.Hardware.PciDevice {
		if strings.Contains(dev.VendorName, devname) {
			targetDev = dev
			log.Printf("Found target device with name: '%v' and model: '%v'", dev.VendorName, dev.DeviceName)
		}
	}
	// hostpciconfigitem := types.HostPciPassthruConfig{Id: targetDev.Id, PassthruEnabled: true}
	// passthroughConfig := []types.BaseHostPciPassthruConfig{hostpciconfigitem.GetHostPciPassthruConfig()}
	// req := types.UpdatePassthruConfig{
	// 	This:   passthroughSystem.Reference(),
	// 	Config: passthroughConfig,
	// }
	req := types.UpdatePassthruConfig{
		This: passthroughSystem.Reference(),
		Config: []types.BaseHostPciPassthruConfig{
			&types.HostPciPassthruConfig{
				Id:              targetDev.Id,
				PassthruEnabled: true,
			},
		},
	}

	res, err := methods.UpdatePassthruConfig(ctx, client.RoundTripper, &req)
	if err != nil {
		return err
	}
	log.Printf("Config response: %+v", res)

	return nil
}

func (sdc *SDCESXi) deployVnics(c *govmomi.Client) error {

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Connect and login to ESX or vCenter
	c, err := examples.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Logout(ctx)

	//create a property collector to use later
	pc := property.DefaultCollector(c.Client)

	// Get all the DVS objects
	m := view.NewManager(c.Client)

	v, err := m.CreateContainerView(ctx, c.ServiceContent.RootFolder, []string{"DistributedVirtualSwitch"}, true)
	if err != nil {
		log.Fatal(err)
	}

	//get the network manager for this host
	networkSystemobj, err := sdc.HostSystem.ConfigManager().NetworkSystem(ctx)
	if err != nil {
		return err
	}

	defer v.Destroy(ctx)
	var dvss []mo.DistributedVirtualSwitch

	err = v.Retrieve(ctx, []string{"DistributedVIrtualSwitch"}, nil, &dvss)

	val := reflect.ValueOf(sdc.Network).Elem()

	for i := 0; i < val.NumField(); i++ {
		valueField := val.Field(i)
		typeField := val.Type().Field(i)

		targetPortgroup := typeField.Name
		var network Network
		network = valueField.Interface().(Network)
		var hostVirtualNicSpec types.HostVirtualNicSpec
		if network.Dvs != "" {
			dvsfound := false
			for _, dvs := range dvss {

				if dvs.Name != network.Dvs {
					//wrong DVS, try next one.
					continue
				}
				dvsfound = true
				cfg := dvs.Config.GetDVSConfigInfo()
				hostfound := false
				for _, host := range cfg.Host {
					if host.Config.Host.Reference() == sdc.HostSystem.Reference() {
						hostfound = true
					}
				}
				if hostfound == false {
					log.Printf("Host: '%v' is not found on DVS: '%v'", sdc.HostSystem.Name(), dvs.Name)
					return fmt.Errorf("Host: '%v' is not found on DVS: '%v'", sdc.HostSystem.Name(), dvs.Name)
				}
				portgroupfound := false
				var portgroups []mo.DistributedVirtualPortgroup
				err = pc.Retrieve(ctx, dvs.Portgroup, []string{}, &portgroups)
				if err != nil {
					return err
				}
				for _, portgroup := range portgroups {

					if portgroup.Name != targetPortgroup { //we may need to find a better data construct as '-' isn't allowed in struct attribute names
						continue
					}
					distributedVirtualPort := types.DistributedVirtualSwitchPortConnection{
						PortgroupKey: portgroup.Key,
						SwitchUuid:   dvs.Uuid,
					}
					hostVirtualNicSpec = types.HostVirtualNicSpec{Ip: &types.HostIpConfig{
						Dhcp:       false,
						IpAddress:  network.IP,
						SubnetMask: network.Netmask},
						DistributedVirtualPort: &distributedVirtualPort,
					}
					portgroupfound = true
				}
				if portgroupfound == false {
					return fmt.Errorf("Portgroup '%v' not found on DVS '%v'", targetPortgroup, dvs.Name)
				}
				result, err := networkSystemobj.AddVirtualNic(ctx, "", hostVirtualNicSpec)
				if err != nil {
					log.Printf("Error adding VNIC '%v' to vSwitch", targetPortgroup+"-vnic")
					return err
				}
				log.Printf("Virtual NIC response: %v", result)
			}
			if dvsfound == false {
				return fmt.Errorf("DVS: '%v' was not found on this vCenter", network.Dvs)
			}
			if network.Vswitch != "" {
				var networkSystems []mo.HostNetworkSystem
				err = pc.Retrieve(ctx, []types.ManagedObjectReference{networkSystemobj.Reference()}, []string{}, &networkSystems)
				if err != nil {
					return err
				}
				networkSystem := networkSystems[0]
				portgroupfound := false
				for _, portgroup := range networkSystem.NetworkInfo.Portgroup {
					if portgroup.Spec.Name != targetPortgroup {
						continue
					}
					//vnicfound := false
					for _, vnic := range networkSystem.NetworkInfo.Vnic {
						if vnic.Portgroup == targetPortgroup {
							log.Printf("VNIC already exists on portgroup '%v'", vnic.Portgroup)
							networkSystemobj.RemoveVirtualNic(ctx, vnic.Device)
						}
						if vnic.Portgroup == targetPortgroup+"-vnic" {
							log.Printf("VNIC already exists on portgroup '%v'", vnic.Portgroup)
							networkSystemobj.RemoveVirtualNic(ctx, vnic.Device)
							networkSystemobj.RemovePortGroup(ctx, targetPortgroup+"-vnic")
						}
					}
					networkSystemobj.AddPortGroup(ctx,
						types.HostPortGroupSpec{Name: targetPortgroup + "-vnic",
							VlanId:      network.Vlan,
							VswitchName: network.Vswitch,
							Policy:      types.HostNetworkPolicy{}},
					)
					hostVirtualNicSpec = types.HostVirtualNicSpec{Ip: &types.HostIpConfig{
						Dhcp:       false,
						IpAddress:  network.IP,
						SubnetMask: network.Netmask},
					}
					portgroupfound = true
				}
				if portgroupfound == false {
					fmt.Errorf("Portgroup not found '%v' on vSwitch", targetPortgroup)
				}
				result, err := networkSystemobj.AddVirtualNic(ctx, targetPortgroup+"-vnic", hostVirtualNicSpec)
				if err != nil {
					log.Printf("Error adding VNIC '%v' to DVS", targetPortgroup+"-vnic")
					return err
				}
				log.Printf("Virtual NIC response: %v", result)
			}

		}
	}
	return nil
}

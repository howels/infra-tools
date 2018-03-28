package main

import (
	"context"
	"log"

	"github.com/howels/infra-tools/vsphere"
	"github.com/vmware/govmomi/examples"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
)

func main() {

	// Currently using env variables from
	// GOVMOMI_URL
	// GOVMOMI_USERNAME
	// GOVMOMI_PASSWORD
	// GOVMOMI_INSECURE

	var fpath = "/home/howels/Downloads/VMware-VCSA-all-6.0.0-5326177.ova"
	//var fpath = "*.ovf"
	var optpath = "/home/howels/vcsa-completed.json"
	var vcName = "test-vc.int.vcevlab.com"
	var vmName = "test-vc"
	var datastoreName = "vol-SSD_TOSHIBA_PX04SMB080-1"
	var clusterName = "c3x1"
	var datacenterName = "/c3x1"
	var network = []vsphere.Network{vsphere.Network{Name: "Network 1", Network: "APP1"}}
	var options = &vsphere.OptionsFlag{
		Target: &vsphere.OptionsFlagVC{
			DatacenterName: datacenterName,
			DatastoreName:  datastoreName,
			ClusterName:    clusterName,
		},
		Path: fpath,
		Options: vsphere.Options{
			NetworkMapping: network,
			Name:           &vcName,
		},
	}

	// Simple example code to login
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	//Parse input JSON into options object
	err := options.ProcessOptionsFile(ctx, optpath)
	if err != nil {
		log.Print("Failed to parse options")
	}
	log.Printf("%+v", options)

	// Connect and log in to ESX or vCenter
	c, err := examples.NewClient(ctx)
	if err != nil {
		log.Fatal(err)
	}

	defer c.Logout(ctx)

	f := find.NewFinder(c.Client, true)

	// Find one and only datacenter
	dc, err := f.DefaultDatacenter(ctx)
	if err != nil {
		panic(err)
	}

	// Make future calls local to this datacenter
	f.SetDatacenter(dc)

	// Find virtual machines in datacenter
	vms, err := f.VirtualMachineList(ctx, "*")
	if err != nil {
		panic(err)
	}
	var vm *object.VirtualMachine
	for _, a := range vms {
		if a.Name() == vmName {
			vm = a
		}
	}
	if vm != nil {
		task, err := vm.PowerOff(ctx)
		if err != nil {
			log.Print(err)
		}

		if _, err = task.WaitForResult(ctx, nil); err != nil {
			log.Print(err)
		}

		task, err = vm.Destroy(ctx)
		if err != nil {
			panic(err)
		}

		if _, err = task.WaitForResult(ctx, nil); err != nil {
			panic(err)
		}
	}

	vsphere.Upload(ctx, options, c)
}

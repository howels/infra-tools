package main

import (
	"context"
	"log"

	"github.com/howels/infra-tools/scaleio"
	"github.com/howels/infra-tools/vsphere"
)

func main() {

	// var pacakgeURL = "http://mlb-repos.ctc.lab.vce.com:8081/nexus/content/repositories/general/ScaleIO_2.0.0.2_RHEL7_Download.zip"
	// var centos = scaleio.Installation{
	// 	PrereqCommands: []string{"yum install -y unzip libaio numactl", fmt.Sprintf("rm -rf /root/install && curl %v > /tmp/sio_pkg.zip && cd /root && unzip /tmp/sio_pkg.zip && mv ScaleIO* install", pacakgeURL)},
	// 	EraseCommands:  []string{"rpm -e $(rpm -qa 'EMC-ScaleIO*')"},
	// }
	// log.Print(centos)

	//sio := &scaleio.ScaleIO{Password: "SIOPass123"}

	configFile := "/home/stephen/go/src/github.com/howels/infra-tools/scaleio/vxrack_config_c3x1_full.yml"
	config := scaleio.Import(configFile)
	//node := &scaleio.Node{}
	log.Printf("Config: %+v", config)
	// panic(nil)
	// log.Printf("Vblock: %v", config.Vblock)

	// Simple example code to login
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// // Connect and log in to ESX or vCenter
	// c, err := examples.NewClient(ctx)
	// if err != nil {
	// 	log.Fatal(err)
	// }

	// defer c.Logout(ctx)

	// f := find.NewFinder(c.Client, true)

	// // Find one and only datacenter
	// dc, err := f.DefaultDatacenter(ctx)
	// if err != nil {
	// 	panic(err)
	// }

	// // Make future calls local to this datacenter
	// f.SetDatacenter(dc)

	// // Find virtual machines in datacenter
	// hosts, err := f.HostSystemList(ctx, "*")
	// if err != nil {
	// 	panic(err)
	// }
	// var h *object.HostSystem
	// for _, a := range hosts {
	// 	if a.Name() == "c3x1-h05.int.vcevlab.com" {
	// 		h = a
	// 	}
	// }

	vcCred := &vsphere.Credentials{IP: "c3x1-vc-cust.int.vcevlab.com",
		User: "administrator@vsphere.local",
		Pass: "Whatever!123"}
	vc := &vsphere.Vcenter{Credentials: vcCred, Insecure: true, Context: ctx}
	hostname := "c3x1-h05.int.vcevlab.com"
	defer vc.Logout()

	h, err := vc.FindHostSystemByName(hostname)
	if err != nil {
		panic(err)
	}

	sdcesxi := &scaleio.SDCESXi{HostSystem: h, Vcenter: vc}
	log.Printf("Host to be used: '%v'", sdcesxi.HostSystem.Name())
	err = sdcesxi.EnablePassthrough("LSI")
	if err != nil {
		panic(err)
	}
}

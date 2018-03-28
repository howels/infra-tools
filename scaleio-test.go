package main

import (
	"fmt"
	"log"

	"github.com/howels/infra-tools/scaleio"
)

func main() {

	var pacakgeURL = "http://mlb-repos.ctc.lab.vce.com:8081/nexus/content/repositories/general/ScaleIO_2.0.0.2_RHEL7_Download.zip"
	var centos = scaleio.Installation{
		PrereqCommands: []string{"yum install -y unzip libaio numactl", fmt.Sprintf("rm -rf /root/install && curl %v > /tmp/sio_pkg.zip && cd /root && unzip /tmp/sio_pkg.zip && mv ScaleIO* install", pacakgeURL)},
		EraseCommands:  []string{"rpm -e $(rpm -qa 'EMC-ScaleIO*')"},
	}
	log.Print(centos)

	sio := &scaleio.ScaleIO{Password: "SIOPass123"}

	// configFile := "/home/stephen/go/src/github.com/howels/infra-tools/scaleio/vxrack_config_c3x1_full.yml"
	// config := scaleio.Import(configFile)
	// //node := &scaleio.Node{}
	// log.Printf("Config: %+v", config)
	// log.Printf("Vblock: %v", config.Vblock)

	commands := []string{"whoami", "ls /"}
	node := scaleio.NewMDMNode("sshtest", "sshtest", "localhost", nil, "127.0.0.1/8", false, sio)
	node.Installation = centos
	data, err := node.Commands(commands)
	if err != nil {
		panic(err)
	}
	for _, d := range data {
		log.Printf("Stdout: %v", d.Stdout)
		log.Printf("Stderr: %v", d.Stderr)
	}
}

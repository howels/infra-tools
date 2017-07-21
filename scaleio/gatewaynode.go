package scaleio

import (
	"net"

	"fmt"

	"github.com/howels/infra-tools/ssh"
)

//GatewayNode describes the properties of the VM to be built
type GatewayNode struct {
	*Node
	ScaleIO *ScaleIO
}

//NewGatewayNode passes a new node object
func NewGatewayNode(Username string, Password string, Hostname string, DataCIDR []string, ManagementCIDR string, UseSudo bool, ScaleIO *ScaleIO) *GatewayNode {

	ip, _, err := net.ParseCIDR(ManagementCIDR)
	if err != nil {
		panic(err)
	}
	sshClient := sshclient.NewSSHClient(Username, Password, ip.String())
	var Become string
	if UseSudo {
		Become = "sudo bash -c "
	} else {
		Become = "bash -c "
	}
	node := &Node{SSH: sshClient,
		Hostname:          Hostname,
		DataNetworks:      DataCIDR,
		ManagementNetwork: ManagementCIDR,
		Become:            Become,
	}
	node.Installation.InstallCommands = []string{"yum install java-1.8.0-openjdk-headless",
		"export SIO_GW_JAVA=/usr/java/default",
		fmt.Sprintf("export GATEWAY_ADMIN_PASSWORD=%v", ScaleIO.Password),
		"cd /root/install && rpm -i EMC-ScaleIO-gateway-*.rpm"}
	return &GatewayNode{Node: node, ScaleIO: ScaleIO}
}

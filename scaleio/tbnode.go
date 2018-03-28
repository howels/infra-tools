package scaleio

import (
	"net"

	"github.com/howels/infra-tools/ssh"
)

//TBNode describes the properties of the VM to be built
type TBNode struct {
	*Node
	ScaleIO *ScaleIO
}

//NewTBNode passes a new node object
func NewTBNode(Username string, Password string, Hostname string, DataCIDR []string, ManagementCIDR string, UseSudo bool, ScaleIO *ScaleIO) *TBNode {

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
	node.Installation.InstallCommands = []string{"cd /root/install && MDM_ROLE_IS_MANAGER=0 rpm -i EMC-ScaleIO-mdm-*.rp"}
	return &TBNode{Node: node, ScaleIO: ScaleIO}
}

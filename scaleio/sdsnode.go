package scaleio

import (
	"net"

	"github.com/howels/infra-tools/ssh"
)

//SDSNode describes the properties of the VM to be built
type SDSNode struct {
	*Node
}

//NewSDSNode passes a new node object
func NewSDSNode(Username string, Password string, Hostname string, DataCIDR []string, ManagementCIDR string, UseSudo bool) *SDSNode {

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
	node.Installation.InstallCommands = []string{"cd /root/install && rpm -i EMC-ScaleIO-sds-*.rpm EMC-ScaleIO-lia-*.rpm"}
	return &SDSNode{Node: node}
}

//Install sets up the VM
func (node *SDSNode) Install() error {
	node.Commands(node.Installation.PrereqCommands)
	return nil
}

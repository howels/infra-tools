package scaleio

import (
	"fmt"
	"log"
	"net"

	"github.com/howels/infra-tools/ssh"
)

//MDMNode describes the properties of the VM to be built
type MDMNode struct {
	*Node
	ScaleIO *ScaleIO
}

//NewMDMNode passes a new node object
func NewMDMNode(Username string, Password string, Hostname string, DataCIDR []string, ManagementCIDR string, UseSudo bool, ScaleIO *ScaleIO) *MDMNode {

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
	node.Installation.InstallCommands = []string{"cd /root/install && MDM_ROLE_IS_MANAGER=1 rpm -i EMC-ScaleIO-mdm-*.rpm"}
	return &MDMNode{Node: node, ScaleIO: ScaleIO}
}

func (mdm *MDMNode) login() error {
	loginCommand := fmt.Sprintf("scli --mdm_ip=%v --login --username admin --password %v", mdm.DataIPString(), mdm.ScaleIO.Password)
	output, err := mdm.Command(loginCommand)
	if err != nil {
		return err
	}
	log.Printf("Login success: %v", output.Stdout)
	return nil

}

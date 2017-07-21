package scaleio

import (
	"fmt"
	"net"
	"strings"

	"github.com/howels/infra-tools/ssh"
)

// SecureShellClient describes the basic connection parameters
type SecureShellClient struct {
	username string
	password string
	hostname string
	become   string //the command to obtain root account if required.
}

// //IPAddresses describes the node's IP
// type IPAddresses struct {
// 	dataIP       []IPNet
// 	managementIP IPNet
// }

//Installation describes commands for install
type Installation struct {
	PrereqCommands  []string
	InstallCommands []string
	EraseCommands   []string
	SioPackageURL   string
}

//InstallationManager is intended to be an opportunity for DI of installation methods
type InstallationManager interface {
	Install() error
}

//Node describes the properties of the VM to be built
type Node struct {
	SSH               sshclient.ShellConnection
	Installation      Installation
	DataNetworks      []string
	ManagementNetwork string
	Hostname          string
	Become            string
}

//NewNode passes a new node object
func NewNode(Username string, Password string, Hostname string, DataCIDR []string, ManagementCIDR string, UseSudo bool) *Node {

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
	return &Node{SSH: sshClient,
		Hostname:          Hostname,
		DataNetworks:      DataCIDR,
		ManagementNetwork: ManagementCIDR,
		Become:            Become,
	}
}

//Command executes an SSH command
func (node *Node) Command(cmd string) (*sshclient.CommandOutput, error) {
	// s := sshclient.NewSSHClient("sshtest", "sshtest", "localhost")
	out, err := node.SSH.Command(node.Become + "\"" + cmd + "\"")
	if err != nil {
		return out, err
	}
	fmt.Println(string(out.Stdout))
	return out, nil
}

//Commands executes a list of commands
func (node *Node) Commands(cmds []string) ([]*sshclient.CommandOutput, error) {
	var output []*sshclient.CommandOutput
	for _, cmd := range cmds {
		out, err := node.SSH.Command(cmd)
		output = append(output, out)
		if err != nil {
			return output, err
		}
	}
	return output, nil
}

//Install sets up the VM
func (node *Node) Install() error {
	_, err := node.Commands(node.Installation.PrereqCommands)
	if err != nil {
		return err
	}
	_, err = node.Commands(node.Installation.EraseCommands)
	if err != nil {
		return err
	}
	_, err = node.Commands(node.Installation.InstallCommands)
	if err != nil {
		return err
	}
	return nil
}

//MgmtIP produces the IP needed to connect.
func (node *Node) MgmtIP() net.IP {
	ip, _, err := net.ParseCIDR(node.ManagementNetwork)
	if err != nil {
		panic(err)
	}
	return ip
}

//MgmtIPString provides the CSV IP string the ScaleIO uses
func (node *Node) MgmtIPString() string {
	return node.MgmtIP().String()
}

//DataIPString is a commaseparated list of IPs for this node
func (node *Node) DataIPString() string {
	var ips []string
	for _, cidr := range node.DataNetworks {
		ip, _, err := net.ParseCIDR(cidr)
		if err != nil {
			panic(err)
		}
		ips = append(ips, ip.String())
	}
	return strings.Join(ips, ",")
}

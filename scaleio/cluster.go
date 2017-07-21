package scaleio

import (
	"fmt"
	"log"
	"strings"

	"github.com/howels/infra-tools/ssh"
)

//Cluster describes the properties clusttered MDM/TB group.
type Cluster struct {
	MDMs      []*MDMNode
	TBs       []*TBNode
	SDSs      []*SDSNode
	ScaleIO   *ScaleIO
	IsCluster bool
	Options   *clusterOptions
}

type clusterOptions struct {
	NumberMDM int
	NumberTB  int
}

//Defaults sets certain usual options
func (cluster *Cluster) Defaults() {
	cluster.Options = &clusterOptions{NumberMDM: 2, NumberTB: 1}
}

func (cluster *Cluster) mdmIP() string {
	var output string
	if cluster.IsCluster {
		var ips []string
		for _, mdm := range cluster.MDMs {
			ips = append(ips, mdm.DataIPString())
		}
		output = strings.Join(ips, ",")
	} else {
		output = cluster.MDMs[0].DataIPString()
	}
	return output
}

func (cluster *Cluster) scliIP() string {
	if len(cluster.MDMs) == 0 {
		log.Fatal("No MDMs added to cluster, cannot locate scliIP")
	}
	return cluster.MDMs[0].MgmtIPString()
}

func (cluster *Cluster) command(cmd string) (*sshclient.CommandOutput, error) {
	return cluster.MDMs[0].Command(cmd)
}

func (cluster *Cluster) login() error {
	login := false
	retries := 0
	var err error
	var output *sshclient.CommandOutput
	for login == false && retries < cluster.ScaleIO.MaxRetries {
		retries = retries + 1
		loginCommand := fmt.Sprintf("scli --mdm_ip=%v --login --username admin --password %v", cluster.mdmIP(), cluster.ScaleIO.Password)
		output, err = cluster.command(loginCommand)
		if err != nil {
			cluster.MDMs = rotate(cluster.MDMs, 1)
		}
	}
	if retries == cluster.ScaleIO.MaxRetries {
		log.Printf("Login failed, maximum attempts used: %v", output.Stderr)
		return err
	}
	log.Printf("Login success: %v", output.Stdout)
	return nil
}

func rotate(a []*MDMNode, i int) []*MDMNode {
	x, b := (a)[:i], (a)[i:]
	a = append(b, x...)
	return a
}

func (cluster *Cluster) activateCluster() error {
	if len(cluster.MDMs) < cluster.Options.NumberMDM {
		log.Fatal(fmt.Sprintf("Need more MDM nodes (%v) to activate cluster, found only (%v)", cluster.Options.NumberMDM, len(cluster.MDMs)))
	}
	if len(cluster.TBs) < cluster.Options.NumberTB {
		log.Fatal(fmt.Sprintf("Need more TB nodes (%v)to activate cluster, found only (%v)", cluster.Options.NumberTB, len(cluster.TBs)))
	}
	cluster.login()

	_, err := cluster.command(fmt.Sprintf("scli --mdm_ip='%v' --switch_to_cluster_mode", cluster.mdmIP()))
	if err != nil {
		return err
	}
	_, err = cluster.command(fmt.Sprintf("scli --mdm_ip=%v --switch_cluster_mode --cluster_mode 3_node --add_slave_mdm_name %v --add_tb_name %v", cluster.mdmIP(), cluster.MDMs[1].Hostname, cluster.TBs[0].Hostname))
	if err != nil {
		return err
	}
	cluster.IsCluster = true
	return nil
}

//SetPassword sets the ScaleIO password
func (cluster *Cluster) SetPassword(password string) error {
	_, err := cluster.command(fmt.Sprintf("scli --login --username admin --password %v", cluster.ScaleIO.Password))
	if err != nil {
		return err
	}
	_, err = cluster.command(fmt.Sprintf("scli --set_password --old_password %v --new_password %v", cluster.ScaleIO.Password, password))
	if err != nil {
		return err
	}
	err = cluster.login()
	if err != nil {
		return err
	}
	cluster.ScaleIO.Password = password
	return nil
}

//AddMDMStandby adds the other MDM node
func (cluster *Cluster) AddMDMStandby(mdm MDMNode) error {
	err := cluster.login()
	if err != nil {
		return err
	}
	_, err = cluster.command(fmt.Sprintf("scli --mdm_ip=%v --add_standby_mdm --new_mdm_ip %v --mdm_role manager --new_mdm_management_ip %v --new_mdm_name %v", cluster.mdmIP(), mdm.DataIPString(), mdm.DataIPString(), mdm.Hostname))
	if err != nil {
		return err
	}
	return nil
}

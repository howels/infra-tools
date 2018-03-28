package scaleio

import (
	"fmt"
)

//ScaleIO describes the properties of the overall ScaleIO environment
type ScaleIO struct {
	Password   string //defaults to 'admin' on initial install
	MaxRetries int
}

//NewCluster passes back the new struct and adds the first MDM
func (sio *ScaleIO) NewCluster(mdm *MDMNode) (*Cluster, error) {
	err := sio.createClusterCommand(mdm)
	if err != nil {
		return nil, err
	}
	cluster := &Cluster{MDMs: []*MDMNode{mdm}, TBs: []*TBNode{}, SDSs: []*SDSNode{}, ScaleIO: sio, IsCluster: false}
	cluster.Defaults()
	return cluster, nil
}

func (sio *ScaleIO) createClusterCommand(mdm *MDMNode) error {
	createClusterCommand := fmt.Sprintf("scli --mdm_ip=%v --create_mdm_cluster --master_mdm_ip %v --master_mdm_management_ip %v --master_mdm_name %v --accept_license --approve_certificate", mdm.DataIPString(), mdm.DataIPString(), mdm.MgmtIPString(), mdm.Hostname)
	_, err := mdm.Command(createClusterCommand)
	return err
}

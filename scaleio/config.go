package scaleio

import (
	"io/ioutil"
	"log"

	// yaml "gopkg.in/yaml.v2"
	"github.com/ghodss/yaml"
)

//BMC credentials
type BMC struct {
	username string
	password string
	hostname string
}

//Config is thre top-level item
type Config struct {
	Vblock         string
	VCSAManagement ConfigVM `json:"vcsa_management"`
	VCSACustomer   ConfigVM `json:"vcsa_customer"`
	Switches       []Credentials
	Hosts          []Host `json:"hosts"`
	OS             string `json:"os_name"`
	Vlans          struct {
		Trunk  []string
		Native string
	}
	DVSPortgroups map[string]string `json:"dvs_portgroups"`
	OVA           string            `json:"ova"`
	DNSSuffixes   []string          `json:"dns_suffixes"`
	DNSServers    []string          `json:"dns_servers"`
	Domain        string
	Datacenter    string
	VibURL        string `json:"vib_url"`
	PackageURL    string `json:"package_url"`
}

//ConfigVM carrries the location where an OVA is installed plus it's config parameters
type ConfigVM struct {
	Target Target
	Config struct {
		VCSA ConfigVCSA `json:"vcsa"`
	}
}

//Target descibes the destination where an OVA is deployed.
type Target struct {
	ESXi           Credentials `json:"esxi"`
	Portgroup      string
	PortgroupScope string `json:"portgroup_scope"`
	PortgroupVLAN  string `json:"portgroup_vlan"`
	Datastore      string
	Vcenter        Credentials `json:",omitempty"`
}

//ConfigVCSA carries VCSA-specific parameters for building the VM.
type ConfigVCSA struct {
	OVA              string `json:"ova"`
	FQDN             string `json:"fqdn"`
	IP               string `json:"ip"`
	prefix           string
	gateway          string
	DNS              string `json:"dns"`
	password         string
	DeploymentOption string
	DiskMode         string
}

//Credentials is the minimum login info for equipment using username/password auth
type Credentials struct {
	IP   string
	User string
	Pass string
}

//Host describes an (ESXi) host to be built
type Host struct {
	Hostname  string   `json:"hostname"`
	Location  []string `json:"location"`
	Mac       string   `json:",omitempty"`
	BMCIP     string   `json:"bmc_ip"`
	BMCUser   string   `json:"bmc_user"`
	BMCPass   string   `json:"bmc_pass"`
	Datastore string
	Network   NetworkMap
	SDS       SDSConfig `json:"sds"`
}

//NetworkMap is a list of ScaleIO-specific networks
type NetworkMap struct {
	SIODATA1 Network `json:"SIO-DATA1"`
	SIODATA2 Network `json:"SIO-DATA2"`
	SIOMGMT  Network `json:"SIO-MGMT,omniempty"`
}

//Network is a ScaleIO network object as used on ESXi hosts and SDS Vms
type Network struct {
	Role     string
	Nics     []string `json:",omitempty"`
	Vswitch  string   `json:",omitempty"`
	Dvs      string   `json:",omitempty"`
	Teaming  string   `json:",omitempty"`
	IP       string   `json:"ip"`
	Netmask  string
	Gateway  string `json:",omitempty"`
	Vlan     int32  `json:",omitempty"`
	VnicName string `json:"vnic_name,omitempty"`
}

//SDSConfig describes an SDS VM
type SDSConfig struct {
	VMName          string `json:"vm_name"`
	NetworkMappings struct {
		Nat string
	} `json:"network_mappings"`
	Network NetworkMap
}

//Import takes the config file
func Import(path string) *Config {

	//filename = "vxrack"
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatal(err)
	}
	//log.Printf("Input data:\n %v", string(data))
	config := Config{}
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Fatal(err)
	}

	return &config

}

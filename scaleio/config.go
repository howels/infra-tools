package scaleio

import (
	"io/ioutil"
	"log"

	// yaml "gopkg.in/yaml.v2"
	"github.com/ghodss/yaml"
)

type BMC struct {
	username string
	password string
	hostname string
}

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

type ConfigVM struct {
	Target Target
	Config struct {
		VCSA ConfigVCSA `json:"vcsa"`
	}
}

type Target struct {
	ESXi           Credentials `json:"esxi"`
	Portgroup      string
	PortgroupScope string `json:"portgroup_scope"`
	PortgroupVLAN  string `json:"portgroup_vlan"`
	Datastore      string
	Vcenter        Credentials `json:",omitempty"`
}

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

type Credentials struct {
	IP   string
	User string
	Pass string
}

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

type NetworkMap struct {
	SIODATA1 Network `json:"SIO-DATA1"`
	SIODATA2 Network `json:"SIO-DATA2"`
	SIOMGMT  Network `json:"SIO-MGMT,omniempty"`
}

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

package vsphere

import (
	"context"
	"encoding/json"
	"flag"
	"os"

	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25/types"
	//"github.com/vmware/govmomi/vim25/types"
)

// Property is one of the OVF keys used to supply parameters to the new VM
type Property struct {
	types.KeyValue
	Spec *ovf.Property `json:",omitempty"`
}

// Network is a vSphere network object
type Network struct {
	Name    string
	Network string
}

// Options object in vSphere OVF deployment for use as argument
type Options struct {
	AllDeploymentOptions []string `json:",omitempty"`
	Deployment           string

	AllDiskProvisioningOptions []string `json:",omitempty"`
	DiskProvisioning           string

	AllIPAllocationPolicyOptions []string `json:",omitempty"`
	IPAllocationPolicy           string

	AllIPProtocolOptions []string `json:",omitempty"`
	IPProtocol           string

	PropertyMapping []Property `json:",omitempty"`

	NetworkMapping []Network `json:",omitempty"`

	Annotation string `json:",omitempty"`

	PowerOn      bool
	InjectOvfEnv bool
	WaitForIP    bool
	Name         *string
}

// OptionsFlag carries the path to the JSON file with options data
type OptionsFlag struct {
	Options Options

	Path string
}

func newOptionsFlag(ctx context.Context) (*OptionsFlag, context.Context) {
	return &OptionsFlag{}, ctx
}

// Register adds the filename to output log data (redundant)
func (flag *OptionsFlag) Register(ctx context.Context, f *flag.FlagSet) {
	f.StringVar(&flag.Path, "options", "", "Options spec file path for VM deployment")
}

// Process opens the options file and converts JSON into an options datastructure
func (flag *OptionsFlag) Process(ctx context.Context) error {
	if len(flag.Path) == 0 {
		return nil
	}

	var err error
	in := os.Stdin

	if flag.Path != "-" {
		in, err = os.Open(flag.Path)
		if err != nil {
			return err
		}
		defer in.Close()
	}

	return json.NewDecoder(in).Decode(&flag.Options)
}

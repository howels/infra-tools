package vsphere

import (
	"context"
	"encoding/json"
	"log"
	"os"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
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

// OptionsFlagTarget is common parent of ESXi and VC-based imports
type OptionsFlagTarget interface {
	Validate(context.Context, *vim25.Client) (bool, error)
	Datastore() *object.Datastore
	ResourcePool() *object.ResourcePool
	HostSystem() *object.HostSystem
	Folder() *object.Folder
}

// OptionsFlag is the common parent struct for options
type OptionsFlag struct {
	Options Options
	Target  OptionsFlagTarget
	Path    string
}

// OptionsFlagVC carries the config for VC-based targets
type OptionsFlagVC struct {
	DatastoreName    string
	datastore        *object.Datastore
	DatacenterName   string
	datacenter       *object.Datacenter
	ClusterName      string
	cluster          *object.ClusterComputeResource
	HostName         string
	hostSystem       *object.HostSystem
	FolderName       string
	folder           *object.Folder
	ResourcePoolName string
	resourcePool     *object.ResourcePool
}

// OptionsFlagHost carries the config for ESXi-based deployments
type OptionsFlagHost struct {
	DatastoreName    string
	datastore        *object.Datastore
	hostSystem       *object.HostSystem
	FolderName       string
	folder           *object.Folder
	ResourcePoolName string
	resourcePool     *object.ResourcePool
}

//Datastore gets the datastore object
func (flag *OptionsFlagVC) Datastore() *object.Datastore {
	return flag.datastore
}

//Datastore gets the datastore object
func (flag *OptionsFlagHost) Datastore() *object.Datastore {
	return flag.datastore
}

//ResourcePool get the resource pool object
func (flag *OptionsFlagVC) ResourcePool() *object.ResourcePool {
	return flag.resourcePool
}

//ResourcePool get the resource pool object
func (flag *OptionsFlagHost) ResourcePool() *object.ResourcePool {
	return flag.resourcePool
}

//HostSystem get the HostSystem object
func (flag *OptionsFlagVC) HostSystem() *object.HostSystem {
	return flag.hostSystem
}

//HostSystem get the HostSystem object
func (flag *OptionsFlagHost) HostSystem() *object.HostSystem {
	return flag.hostSystem
}

//Folder get the Folder object
func (flag *OptionsFlagVC) Folder() *object.Folder {
	return flag.folder
}

//Folder get the Folder object
func (flag *OptionsFlagHost) Folder() *object.Folder {
	return flag.folder
}

//Validate checks options
func (flag *OptionsFlagVC) Validate(ctx context.Context, client *vim25.Client) (bool, error) {
	//Locate datacenter
	finder := find.NewFinder(client, true)
	// assume we're only using 1 datacenter for now
	datacenter, err := finder.DefaultDatacenter(ctx)
	if err != nil {
		log.Print("More than one Datacenter was found but code assuming only a single DC present")
		panic(err)
	}
	flag.datacenter = datacenter
	finder = finder.SetDatacenter(datacenter)

	var rootPath = datacenter.InventoryPath
	log.Print("Using the root path: ", rootPath)

	//Locate resource pool
	clusters, err := finder.ClusterComputeResourceList(ctx, "*")
	if err != nil {
		log.Print("Could not retrieve ClusterComputeResource under path: ", rootPath)
		panic(err)
	}
	if len(clusters) == 0 {
		log.Fatal("No clusters found, aborting.")
	}
	var cluster = clusters[0]
	flag.cluster = cluster
	var hosts []*object.HostSystem
	hosts, err = cluster.ComputeResource.Hosts(ctx)
	if err != nil {
		panic(err)
	}
	if len(hosts) == 0 {
		log.Fatal("No hostssystems found, aborting.")
	}
	var host = hosts[0]
	flag.hostSystem = host
	log.Print("Found a cluster called: ", cluster.Name())
	log.Print("Found a host in the cluster called: ", host.Name())
	flag.resourcePool, err = cluster.ComputeResource.ResourcePool(ctx)
	if err != nil {
		log.Print("More than one ResourcePool was found but code assuming only a single ResourcePool present")
		panic(err)
	}
	log.Print("Found a ResourcePool with name: ", flag.resourcePool.Name())

	//Locate datastores
	datastore, err := finder.Datastore(ctx, rootPath+"/datastore/"+flag.DatastoreName)
	// datastores, err := finder.DatastoreList(ctx, "*")
	if err != nil {
		log.Fatal(err)
	}
	// for _, ds := range datastores {
	// 	log.Printf("Datastore: %v %v\n", ds.Name(), ds.InventoryPath)
	// }
	log.Print("Found the datastore with name: ", datastore.Name())

	flag.datastore = datastore

	//log.Print(cmd.NetworkMap(e))

	//Locate folder
	folder, err := finder.Folder(ctx, rootPath+"/vm")
	if err != nil {
		log.Fatal("Cannot find VM folder: ", err)
	}
	flag.folder = folder

	//Locate networks
	//TODO
	return true, nil
}

//Validate checks options
func (flag *OptionsFlagHost) Validate(ctx context.Context, client *vim25.Client) (bool, error) {
	//Locate datacenter
	finder := find.NewFinder(client, true)
	// assume we're only using 1 datacenter for now
	var rootPath = "/"
	log.Print("Using the root path: ", rootPath)

	host, err := finder.DefaultHostSystem(ctx)
	if err != nil {
		panic(err)
	}

	flag.hostSystem = host

	log.Print("Found a host in the cluster called: ", host.Name())
	flag.resourcePool, err = host.ResourcePool(ctx)
	if err != nil {
		log.Print("More than one ResourcePool was found but code assuming only a single ResourcePool present")
		panic(err)
	}
	log.Print("Found a ResourcePool with name: ", flag.resourcePool.Name())

	//Locate datastores
	datastore, err := finder.Datastore(ctx, rootPath+"/datastore/"+flag.DatastoreName)
	// datastores, err := finder.DatastoreList(ctx, "*")
	if err != nil {
		log.Fatal(err)
	}
	// for _, ds := range datastores {
	// 	log.Printf("Datastore: %v %v\n", ds.Name(), ds.InventoryPath)
	// }
	log.Print("Found the datastore with name: ", datastore.Name())

	flag.datastore = datastore

	//log.Print(cmd.NetworkMap(e))

	//Locate folder
	folder, err := finder.Folder(ctx, rootPath+"/vm")
	if err != nil {
		log.Fatal("Cannot find VM folder: ", err)
	}
	flag.folder = folder

	//Locate networks
	//TODO
	return true, nil
}

// func newOptionsFlag(ctx context.Context) (*OptionsFlag, context.Context) {
// 	return &OptionsFlag{}, ctx
// }

// // Register adds the filename to output log data (redundant)
// func (flag *OptionsFlag) Register(ctx context.Context, f *flag.FlagSet) {
// 	f.StringVar(&flag.Path, "options", "", "Options spec file path for VM deployment")
// }

// ProcessOptionsFile opens the options file and converts JSON into an options datastructure
func (flag *OptionsFlag) ProcessOptionsFile(ctx context.Context, path string) error {
	if len(path) == 0 {
		return nil
	}

	var err error
	in := os.Stdin

	if path != "-" {
		in, err = os.Open(path)
		if err != nil {
			panic(err)
		}
		defer in.Close()
	}
	log.Print("Options file path: ", path)
	//log.Printf("Options existing: %+v\n", flag.Options)

	json.NewDecoder(in).Decode(&flag.Options)
	//log.Printf("Options decode: %+v\n", &flag.Options)
	return nil
}

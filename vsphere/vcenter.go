package vsphere

import (
	"context"
	"log"
	"net/url"

	"github.com/vmware/govmomi"
	//. "github.com/vmware/govmomi/govc/importx"
	"fmt"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/vim25/soap"
)

//Vcenter is the main struct
type Vcenter struct {
	Credentials *Credentials //move this to a common package
	Insecure    bool
	Client      *govmomi.Client
	Context     context.Context
}

//Credentials holds the connections information
type Credentials struct {
	IP   string
	User string
	Pass string
}

func (vc *Vcenter) envURL() string {
	return fmt.Sprintf("https://%v/sdk", vc.Credentials.IP)
}

func (vc *Vcenter) newClient(ctx context.Context) (*govmomi.Client, error) {
	// ctx, cancel := context.WithCancel(context.Background())
	// defer cancel()

	// Parse URL from string
	u, err := soap.ParseURL(vc.envURL())
	if err != nil {
		return nil, err
	}

	u.User = url.UserPassword(vc.Credentials.User, vc.Credentials.Pass)

	// Connect and log in to ESX or vCenter
	return govmomi.NewClient(ctx, u, vc.Insecure)
}

//Login authenticates with vCenter if not already authenticated
func (vc *Vcenter) Login() error {
	log.Print("Login requested")
	if vc.Client != nil {
		log.Printf("Already logged in")
		return nil
	}
	log.Printf("Login Context: '%v'", vc.Context)
	client, err := vc.newClient(vc.Context)
	if err != nil {
		return err
	}
	vc.Client = client
	return nil
}

//Logout removes the session with vCenter
func (vc *Vcenter) Logout() error {
	log.Print("Logout from VC")
	err := vc.Client.Logout(vc.Context)
	return err
}

//FindHostSystemByName returns a HostSystem object
func (vc *Vcenter) FindHostSystemByName(hostname string) (*object.HostSystem, error) {
	// Connect and log in to ESX or vCenter
	//c, err := examples.NewClient(ctx)

	err := vc.Login()
	if err != nil {
		return nil, err
	}

	//defer vc.Logout()

	f := find.NewFinder(vc.Client.Client, true)

	// Find one and only datacenter
	dc, err := f.DefaultDatacenter(vc.Context)
	if err != nil {
		return nil, err
	}

	// Make future calls local to this datacenter
	f.SetDatacenter(dc)

	// Find virtual machines in datacenter
	hosts, err := f.HostSystemList(vc.Context, "*")
	if err != nil {
		panic(err)
	}
	var h *object.HostSystem
	for _, a := range hosts {
		if a.Name() == hostname {
			h = a
		}
	}
	return h, nil
}

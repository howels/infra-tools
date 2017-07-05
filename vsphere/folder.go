package vsphere

import (
	"context"
	"errors"
	"flag"

	"github.com/vmware/govmomi/govc/flags"
	"github.com/vmware/govmomi/object"
)

// FolderFlag represents folder object in vSphere for use as argument
type FolderFlag struct {
	*flags.DatacenterFlag

	folder string
}

func newFolderFlag(ctx context.Context) (*FolderFlag, context.Context) {
	f := &FolderFlag{}
	f.DatacenterFlag, ctx = flags.NewDatacenterFlag(ctx)
	return f, ctx
}

// Register shows usage info for this command
func (flag *FolderFlag) Register(ctx context.Context, f *flag.FlagSet) {
	flag.DatacenterFlag.Register(ctx, f)

	f.StringVar(&flag.folder, "folder", "", "Path to folder to add the VM to")
}

// Process returns the datacenter item
func (flag *FolderFlag) Process(ctx context.Context) error {
	return flag.DatacenterFlag.Process(ctx)
}

// Folder checks if the requested folder exists
func (flag *FolderFlag) Folder() (*object.Folder, error) {
	ctx := context.TODO()
	if len(flag.folder) == 0 {
		dc, err := flag.Datacenter()
		if err != nil {
			return nil, err
		}
		folders, err := dc.Folders(ctx)
		if err != nil {
			return nil, err
		}
		return folders.VmFolder, nil
	}

	finder, err := flag.Finder()
	if err != nil {
		return nil, err
	}

	mo, err := finder.ManagedObjectList(ctx, flag.folder)
	if err != nil {
		return nil, err
	}
	if len(mo) == 0 {
		return nil, errors.New("folder argument does not resolve to object")
	}
	if len(mo) > 1 {
		return nil, errors.New("folder argument resolves to more than one object")
	}

	ref := mo[0].Object.Reference()
	if ref.Type != "Folder" {
		return nil, errors.New("folder argument does not resolve to folder")
	}

	c, err := flag.Client()
	if err != nil {
		return nil, err
	}

	return object.NewFolder(c, ref), nil
}

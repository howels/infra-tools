package vsphere

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/vmware/govmomi"
	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/govc/flags"
	//. "github.com/vmware/govmomi/govc/importx"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/ovf"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/progress"
	"github.com/vmware/govmomi/vim25/soap"
	"github.com/vmware/govmomi/vim25/types"
)

type ovfx struct {
	//	*flags.DatastoreFlag
	//	*flags.HostSystemFlag
	*flags.OutputFlag
	//	*flags.ResourcePoolFlag

	*ArchiveFlag
	*OptionsFlag
	//  *FolderFlag

	Name string

	Client *vim25.Client
	// Datacenter   *object.Datacenter
	// Datastore    *object.Datastore
	// ResourcePool *object.ResourcePool
	// HostSystem   *object.HostSystem
	// Folder       *object.Folder
	//Archive Archive
	//Options      Options
	//Target       OptionsFlagTarget
}

type ovfFileItem struct {
	url  *url.URL
	item types.OvfFileItem
	ch   chan progress.Report
}

func (o ovfFileItem) Sink() chan<- progress.Report {
	return o.ch
}

type leaseUpdater struct {
	client *vim25.Client
	lease  *object.HttpNfcLease

	pos   int64 // Number of bytes
	total int64 // Total number of bytes

	done chan struct{} // When lease updater should stop

	wg sync.WaitGroup // Track when update loop is done
}

func dectectFileType(filepath string) (string, error) {
	file, err := os.Open(filepath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Only the first 512 bytes are used to sniff the content type.
	buffer := make([]byte, 512)
	_, err = file.Read(buffer)
	if err != nil {
		return "", err
	}

	// Reset the read pointer if necessary.
	file.Seek(0, 0)

	// Always returns a valid content-type and "application/octet-stream" if no others seemed to match.
	contentType := http.DetectContentType(buffer)
	return contentType, nil
}

// Upload is a test function to push a new OVA to vCenter
func Upload(ctx context.Context, optflags *OptionsFlag, c *govmomi.Client) (*object.VirtualMachine, error) {

	// Setup paths and options
	// cli.Register("import.ovf", &ovfx{})
	//var fpath = "/home/howels/Downloads/VMware-VCSA-all-6.0.0-5326177/VMware-vCenter-Server-Appliance-6.0.0.30200-5326079_OVF10.ovf"
	// var fpath = "~/Downloads/VMware-VCSA-all-6.0.0-5326177.ova"
	// //var fpath = "*.ovf"
	//var optpath = "/home/howels/vcsa.json"
	//var vmName = "test-vc.int"
	// var datastoreName = "vol-SSD_TOSHIBA_PX04SMB080-1"

	ftype, err := dectectFileType(optflags.Path)
	if err != nil {
		return nil, err
	}
	var fpath string
	var archive Archive
	if strings.Contains(ftype, "text/xml") {
		// Use FileArchive for OVF
		log.Print("Detected XML, assuming OVF type")
		archive = &FileArchive{Path: optflags.Path}
		fpath = optflags.Path
	} else {
		// Use TapeArchive for OVA
		log.Print("Detected non-XML, assuming OVA type")
		archive = &TapeArchive{Path: optflags.Path}
		fpath = "*.ovf"
	}
	//var options = &Options{Name: &vmName}
	cmd := &ovfx{

		// Name:        *optflags.Options.Name,
		// ArchiveFlag: &ArchiveFlag{Archive: archive},
		// Target:      optflags.Target,
		// Options:    optflags.Options,
		// OutputFlag: &flags.OutputFlag{TTY: true},
		OutputFlag:  &flags.OutputFlag{TTY: true},
		ArchiveFlag: &ArchiveFlag{Archive: archive},
		OptionsFlag: optflags,
		Name:        *optflags.Options.Name,
		Client:      c.Client,
	}

	//cmd.Client = c.Client

	// obtain OVF data and parse
	o, err := cmd.ReadOvf(fpath)
	if err != nil {
		return nil, err
	}

	e, err := cmd.ReadEnvelope(fpath)
	if err != nil {
		log.Fatal(fmt.Errorf("failed to parse ovf: %s", err.Error()))
		return nil, err
	}

	// //parse options
	// err = cmd.OptionsFlag.Process(ctx)
	// if err != nil {
	// 	log.Print("Faileed to parse options")
	// }

	//check the options data
	log.Print("Options flags: ", cmd.Options)
	log.Print("OVF envelope: ", e)
	//log.Print("OVF data: ", o)

	name := "Govc Virtual Appliance"
	if e.VirtualSystem != nil {
		name = e.VirtualSystem.ID
		if e.VirtualSystem.Name != nil {
			name = *e.VirtualSystem.Name
		}
	}

	// Override name from options if specified
	if cmd.Options.Name != nil {
		name = *cmd.Options.Name
	}

	// Override name from arguments if specified
	if cmd.Name != "" {
		name = cmd.Name
	}

	_, err = cmd.Target.Validate(ctx, c.Client)
	if err != nil {
		return nil, err
	}

	log.Printf("Network mapping: %+v", cmd.NetworkMap(e))

	cisp := types.OvfCreateImportSpecParams{
		DiskProvisioning:   cmd.Options.DiskProvisioning,
		EntityName:         name,
		IpAllocationPolicy: cmd.Options.IPAllocationPolicy,
		IpProtocol:         cmd.Options.IPProtocol,
		OvfManagerCommonParams: types.OvfManagerCommonParams{
			DeploymentOption: cmd.Options.Deployment,
			Locale:           "US"},
		PropertyMapping: cmd.Map(cmd.Options.PropertyMapping),
		NetworkMapping:  cmd.NetworkMap(e),
	}

	log.Printf("Ciscp: %+v", cisp)

	ovfm := object.NewOvfManager(cmd.Client)
	spec, err := ovfm.CreateImportSpec(ctx, string(o), cmd.Target.ResourcePool(), cmd.Target.Datastore(), cisp)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}
	if spec.Error != nil {
		log.Fatal(errors.New(spec.Error[0].LocalizedMessage))
	}
	if spec.Warning != nil {
		for _, w := range spec.Warning {
			log.Print(fmt.Sprintf("Warning: %s\n", w.LocalizedMessage))
		}
	}

	if cmd.Options.Annotation != "" {
		switch s := spec.ImportSpec.(type) {
		case *types.VirtualMachineImportSpec:
			s.ConfigSpec.Annotation = cmd.Options.Annotation
		case *types.VirtualAppImportSpec:
			s.VAppConfigSpec.Annotation = cmd.Options.Annotation
		}
	}

	log.Printf("Spec: %+v", spec.ImportSpec)

	lease, err := cmd.Target.ResourcePool().ImportVApp(ctx, spec.ImportSpec, cmd.Target.Folder(), cmd.Target.HostSystem())
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	info, err := lease.Wait(ctx)
	if err != nil {
		log.Fatal(err)
		return nil, err
	}

	// Build slice of items and URLs first, so that the lease updater can know
	// about every item that needs to be uploaded, and thereby infer progress.
	var items []ovfFileItem

	for _, device := range info.DeviceUrl {
		for _, item := range spec.FileItem {

			if device.ImportKey != item.DeviceId {
				continue
			}

			u, err := cmd.Client.ParseURL(device.Url)
			if err != nil {
				log.Fatal(err)
				return nil, err
			}

			i := ovfFileItem{
				url:  u,
				item: item,
				ch:   make(chan progress.Report),
			}

			log.Printf("Adding item '%v' to upload queue", item.Path)
			items = append(items, i)
		}
	}

	u := newLeaseUpdater(cmd.Client, lease, items)
	defer u.Done()

	for _, i := range items {
		err = cmd.FileUpload(lease, i)
		if err != nil {
			log.Fatal(err)
			return nil, err
		}
	}

	moref, err := &info.Entity, lease.HttpNfcLeaseComplete(ctx)
	if err != nil {
		return nil, err
	}
	vm := object.NewVirtualMachine(cmd.Client, *moref)
	err = cmd.Deploy(vm)
	if err != nil {
		return nil, err
	}
	return vm, nil
}

func (cmd *ovfx) Deploy(vm *object.VirtualMachine) error {

	if err := cmd.InjectOvfEnv(vm); err != nil {
		return err
	}

	if err := cmd.PowerOn(vm); err != nil {
		return err
	}

	if err := cmd.WaitForIP(vm); err != nil {
		return err
	}
	return nil
}

func (cmd *ovfx) WaitForIP(vm *object.VirtualMachine) error {
	ctx := context.TODO()
	if !cmd.Options.PowerOn || !cmd.Options.WaitForIP {
		return nil
	}

	cmd.Log("Waiting for IP address...\n")
	ip, err := vm.WaitForIP(ctx)
	if err != nil {
		return err
	}

	cmd.Log(fmt.Sprintf("Received IP address: %s\n", ip))
	return nil
}

func (cmd *ovfx) PowerOn(vm *object.VirtualMachine) error {
	ctx := context.TODO()

	if !cmd.Options.PowerOn {
		return nil
	}

	cmd.Log("Powering on VM...\n")

	task, err := vm.PowerOn(ctx)
	if err != nil {
		return err
	}

	if _, err = task.WaitForResult(ctx, nil); err != nil {
		return err
	}

	return nil
}

func (cmd *ovfx) InjectOvfEnv(vm *object.VirtualMachine) error {
	if !cmd.Options.InjectOvfEnv {
		return nil
	}

	cmd.Log("Injecting OVF environment...\n")

	var opts []types.BaseOptionValue

	a := cmd.Client.ServiceContent.About

	// build up Environment in order to marshal to xml
	var props []ovf.EnvProperty
	for _, p := range cmd.Options.PropertyMapping {
		props = append(props, ovf.EnvProperty{
			Key:   p.Key,
			Value: p.Value,
		})
	}

	env := ovf.Env{
		EsxID: vm.Reference().Value,
		Platform: &ovf.PlatformSection{
			Kind:    a.Name,
			Version: a.Version,
			Vendor:  a.Vendor,
			Locale:  "US",
		},
		Property: &ovf.PropertySection{
			Properties: props,
		},
	}

	opts = append(opts, &types.OptionValue{
		Key:   "guestinfo.ovfEnv",
		Value: env.MarshalManual(),
	})

	ctx := context.Background()

	task, err := vm.Reconfigure(ctx, types.VirtualMachineConfigSpec{
		ExtraConfig: opts,
	})

	if err != nil {
		return err
	}

	return task.Wait(ctx)
}

func (cmd *ovfx) Map(op []Property) (p []types.KeyValue) {
	for _, v := range op {
		p = append(p, v.KeyValue)
	}

	return
}

func (cmd *ovfx) NetworkMap(e *ovf.Envelope) (p []types.OvfNetworkMapping) {
	ctx := context.TODO()
	finder := find.NewFinder(cmd.Client, true)
	if datacenter := cmd.OptionsFlag.Target.Datacenter(); datacenter != nil {
		finder = finder.SetDatacenter(datacenter)
	}

	networks := map[string]string{}

	if e.Network != nil {
		for _, net := range e.Network.Networks {
			networks[net.Name] = net.Name
		}
	}

	for _, net := range cmd.Options.NetworkMapping {
		log.Printf("Mapping network '%v' to portgroup '%v'", net.Name, net.Network)
		networks[net.Name] = net.Network
	}

	for src, dst := range networks {
		if net, err := finder.Network(ctx, dst); err == nil {
			p = append(p, types.OvfNetworkMapping{
				Name:    src,
				Network: net.Reference(),
			})
		} else {
			log.Printf("Cannot locate network: '%v', error: '%v'", dst, err)
		}
	}
	return
}

func (cmd *ovfx) FileUpload(lease *object.HttpNfcLease, ofi ovfFileItem) error {
	item := ofi.item
	file := item.Path

	f, size, err := cmd.Open(file)
	if err != nil {
		return err
	}
	defer f.Close()

	logger := cmd.ProgressLogger(fmt.Sprintf("Uploading %s... ", path.Base(file)))
	defer logger.Wait()

	opts := soap.Upload{
		ContentLength: size,
		Progress:      progress.Tee(ofi, logger),
	}

	// Non-disk files (such as .iso) use the PUT method.
	// Overwrite: t header is also required in this case (ovftool does the same)
	if item.Create {
		opts.Method = "PUT"
		opts.Headers = map[string]string{
			"Overwrite": "t",
		}
	} else {
		opts.Method = "POST"
		opts.Type = "application/x-vnd.vmware-streamVmdk"
	}

	return cmd.Client.Client.Upload(f, ofi.url, &opts)
}

func newLeaseUpdater(client *vim25.Client, lease *object.HttpNfcLease, items []ovfFileItem) *leaseUpdater {
	l := leaseUpdater{
		client: client,
		lease:  lease,

		done: make(chan struct{}),
	}

	for _, item := range items {
		l.total += item.item.Size
		go l.waitForProgress(item)
	}

	// Kickstart update loop
	l.wg.Add(1)
	go l.run()

	return &l
}

func (l *leaseUpdater) waitForProgress(item ovfFileItem) {
	var pos, total int64

	total = item.item.Size

	for {
		select {
		case <-l.done:
			return
		case p, ok := <-item.ch:
			// Return in case of error
			if ok && p.Error() != nil {
				return
			}

			if !ok {
				// Last element on the channel, add to total
				atomic.AddInt64(&l.pos, total-pos)
				return
			}

			// Approximate progress in number of bytes
			x := int64(float32(total) * (p.Percentage() / 100.0))
			atomic.AddInt64(&l.pos, x-pos)
			pos = x
		}
	}
}

func (l *leaseUpdater) run() {
	defer l.wg.Done()

	tick := time.NewTicker(2 * time.Second)
	defer tick.Stop()

	for {
		select {
		case <-l.done:
			return
		case <-tick.C:
			// From the vim api HttpNfcLeaseProgress(percent) doc, percent ==
			// "Completion status represented as an integer in the 0-100 range."
			// Always report the current value of percent, as it will renew the
			// lease even if the value hasn't changed or is 0.
			percent := int32(float32(100*atomic.LoadInt64(&l.pos)) / float32(l.total))
			err := l.lease.HttpNfcLeaseProgress(context.TODO(), percent)
			if err != nil {
				fmt.Printf("from lease updater: %s\n", err)
			}
		}
	}
}

func (l *leaseUpdater) Done() {
	close(l.done)
	l.wg.Wait()
}

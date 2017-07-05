package vsphere

import (
	"archive/tar"
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"

	"github.com/vmware/govmomi/ovf"
)

// ArchiveFlag doesn't register any flags;
// only encapsulates some common archive related functionality.
type ArchiveFlag struct {
	Archive
}

func newArchiveFlag(ctx context.Context) (*ArchiveFlag, context.Context) {
	return &ArchiveFlag{}, ctx
}

// Register comment
func (f *ArchiveFlag) Register(ctx context.Context, fs *flag.FlagSet) {
}

//Process comment
func (f *ArchiveFlag) Process(ctx context.Context) error {
	return nil
}

// ReadOvf retrieves ovf file data
func (f *ArchiveFlag) ReadOvf(fpath string) ([]byte, error) {
	log.Printf("Attempting to open '%v'\n", fpath)
	if f == nil {
		log.Print("Archive is set to nil")
	}
	r, _, err := f.Archive.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	return ioutil.ReadAll(r)
}

// ReadEnvelope retrieves the metadata from the OVF
func (f *ArchiveFlag) ReadEnvelope(fpath string) (*ovf.Envelope, error) {
	if fpath == "" {
		return &ovf.Envelope{}, nil
	}

	r, _, err := f.Open(fpath)
	if err != nil {
		return nil, err
	}
	defer r.Close()

	e, err := ovf.Unmarshal(r)
	if err != nil {
		return nil, fmt.Errorf("failed to parse ovf: %s", err.Error())
	}

	return e, nil
}

// Archive is an interface for DI of input types
type Archive interface {
	Open(string) (io.ReadCloser, int64, error)
}

// TapeArchive used for OVA TAR files
type TapeArchive struct {
	Path string
}

// TapeArchiveEntry used to handle OVA TAR files
type TapeArchiveEntry struct {
	io.Reader
	f *os.File
}

// Close terminates the input stream
func (t *TapeArchiveEntry) Close() error {
	return t.f.Close()
}

// Open retrieves files from within a tar file
func (t *TapeArchive) Open(name string) (io.ReadCloser, int64, error) {
	log.Print("Opening OVA image with path: ", t.Path)
	f, err := os.Open(t.Path)
	if err != nil {
		return nil, 0, err
	}

	r := tar.NewReader(f)

	for {
		h, err := r.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, 0, err
		}
		log.Print("Found tar file entry: ", h.Name)
		matched, err := path.Match(name, path.Base(h.Name))
		if err != nil {
			return nil, 0, err
		}

		if matched {
			return &TapeArchiveEntry{r, f}, h.Size, nil
		}
	}

	_ = f.Close()

	return nil, 0, os.ErrNotExist
}

// FileArchive is used for the normal OVF file loading
type FileArchive struct {
	Path string
}

// Open starts reading from the file
func (t *FileArchive) Open(name string) (io.ReadCloser, int64, error) {
	fpath := name
	if name != t.Path {
		fpath = filepath.Join(filepath.Dir(t.Path), name)
	}

	s, err := os.Stat(fpath)
	if err != nil {
		return nil, 0, err
	}

	f, err := os.Open(fpath)
	if err != nil {
		return nil, 0, err
	}

	return f, s.Size(), nil
}

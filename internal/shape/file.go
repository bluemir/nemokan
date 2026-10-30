package shape

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/bluemir/monosketch/internal/geo"
)

// FileVersion is the version of the File format.
//
// # Version 1
// The first version: name, offset, root, connectors and modifiedTimestampMillis.
const FileVersion = 1

// File is the stored form of a whole document.
type File struct {
	Version                 int                         `json:"version"`
	Name                    string                      `json:"name"`
	Offset                  geo.Point                   `json:"offset"`
	Root                    *SerializableGroup          `json:"root"`
	Connectors              []SerializableLineConnector `json:"connectors"`
	ModifiedTimestampMillis int64                       `json:"modifiedTimestampMillis"`
}

// NewFile creates a File of the current version, modified now.
func NewFile(name string, offset geo.Point, root *SerializableGroup, connectors []SerializableLineConnector) File {
	return File{
		Version:                 FileVersion,
		Name:                    name,
		Offset:                  offset,
		Root:                    root,
		Connectors:              connectors,
		ModifiedTimestampMillis: time.Now().UnixMilli(),
	}
}

// ParseFile reads a File from JSON. A file of an unknown version is an error.
func ParseFile(data []byte) (File, error) {
	var file File
	if err := json.Unmarshal(data, &file); err != nil {
		return File{}, err
	}
	if file.Version != FileVersion {
		return File{}, fmt.Errorf("unknown file version: %d", file.Version)
	}
	if file.Root == nil {
		return File{}, fmt.Errorf("file has no root")
	}
	return file, nil
}

// AsCopy returns the file as a new document, whose root has a new id. The shapes keep their
// ids, so the connectors still point at them.
func (f File) AsCopy() File {
	root := *f.Root
	root.IsIDTemporary = true
	f.Root = RestoreRoot(&root).Serializable(true).(*SerializableGroup)
	return f
}

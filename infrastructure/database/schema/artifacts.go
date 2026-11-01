package schema

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"github.com/golang-migrate/migrate/v4/source"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"os"
	"path/filepath"
)

// artifacts verifies pinned baseline checksums and snapshots bytes before migration can start.
func artifacts(directory string) (source.Driver, error) {
	manifest, err := catalogAssets.ReadFile("checksums.json")
	if err != nil {
		return nil, err
	}
	var sums map[string]string
	if err = json.Unmarshal(manifest, &sums); err != nil {
		return nil, err
	}
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, want := range sums {
		b, e := os.ReadFile(filepath.Join(directory, name))
		if e != nil {
			return nil, e
		}
		sum := sha256.Sum256(b)
		if hex.EncodeToString(sum[:]) != want {
			return nil, errors.New("immutable migration checksum mismatch")
		}
		entry, e := writer.Create(name)
		if e != nil {
			return nil, e
		}
		if _, e = entry.Write(b); e != nil {
			return nil, e
		}
	}
	if err = writer.Close(); err != nil {
		return nil, err
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		return nil, err
	}
	return iofs.New(reader, ".")
}

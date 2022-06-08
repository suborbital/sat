package moduleref

import (
	"encoding/json"
	"io"
	"io/ioutil"

	"github.com/pkg/errors"
)

// WasmModuleRef is a reference to a Wasm module (either its filepath or its bytes)
type WasmModuleRef struct {
	Name   string    `json:"name"`
	FQFN   string    `json:"fqfn"`
	Data   []byte    `json:"data"`
	reader io.Reader `json:"-"`
}

// RefWithData returns a module ref from module bytes
func RefWithData(name, fqfn string, data []byte) *WasmModuleRef {
	ref := &WasmModuleRef{
		Name: name,
		FQFN: fqfn,
		Data: data,
	}

	return ref
}

// RefWithReader returns a module ref from module bytes
func RefWithReader(name, fqfn string, reader io.Reader) *WasmModuleRef {
	ref := &WasmModuleRef{
		Name:   name,
		FQFN:   fqfn,
		reader: reader,
	}

	return ref
}

// Bytes returns the bytes for the module
func (w *WasmModuleRef) Bytes() ([]byte, error) {
	if w.Data == nil {
		if w.reader == nil {
			return nil, errors.New("data not set and no reader available")
		}

		bytes, err := ioutil.ReadAll(w.reader)
		if err != nil {
			return nil, errors.Wrap(err, "failed to ReadAll")
		}

		w.Data = bytes
	}

	return w.Data, nil
}

// MarshalJSON adds a custom marshaller to WasmModuleRef to ensure that
// bytes are read before encoding to JSON
func (w *WasmModuleRef) MarshalJSON() ([]byte, error) {
	if _, err := w.Bytes(); err != nil {
		return nil, errors.Wrap(err, "failed to get Bytes for marshalling")
	}

	return json.Marshal(w)
}

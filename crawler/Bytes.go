package crawler

import (
	"io"
)

/// Lees maximum de eerste 512 bytes van deze reader.
func readFirstBytes(r io.Reader) ([]byte, error) {
	b := make([]byte, 512, 512)
	n, err := r.Read(b)
	if err == io.EOF {
		// done, maar snij onze byte slice bij om lege (niet ingelezen)
		// bytes te verwijderen
		return b[:n], nil
	}

	if err != nil {
		return b, err
	}

	// Er valt nog verder te lezen
	return b, nil
}

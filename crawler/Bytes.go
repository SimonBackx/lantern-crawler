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

	//
	return b, nil
}

/// Leest alle resterende bytes van deze reader. Initialiseer hierbij al met reeds
/// gelezen data uit readFirstBytes
/*func readRemaining(r io.Reader, alreadyRead []byte) (reader io.Reader, err error) {
	buf := bytes.NewBuffer(alreadyRead)
	// If the buffer overflows, we will get bytes.ErrTooLarge.
	// Return that as an error. Any other panic remains.
	defer func() {
		e := recover()
		if e == nil {
			return
		}
		if panicErr, ok := e.(error); ok && panicErr == bytes.ErrTooLarge {
			err = panicErr
		} else {
			panic(e)
		}
	}()
	_, err = buf.ReadFrom(r)

	return bytes.NewReader(buf.Bytes()), err
}*/

package libvirt

import (
	"io"

	"github.com/projecteru2/yavirt/third_party/libvirt/internal/constants"
)

// OpenConsole is the go wrapper for REMOTE_PROC_DOMAIN_OPEN_CONSOLE.
func (l *Libvirt) OpenConsole(Dom Domain, DevName OptString, inStream io.Reader, outStream io.Writer, Flags uint32) (err error) {
	var buf []byte

	args := DomainOpenConsoleArgs{
		Dom:     Dom,
		DevName: DevName,
		Flags:   Flags,
	}

	buf, err = encode(&args)
	if err != nil {
		return
	}

	_, err = l.requestStream(201, constants.Program, buf, inStream, outStream)
	if err != nil {
		return
	}

	return
}

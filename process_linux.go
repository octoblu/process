package process

import (
	"fmt"
	"syscall"
)

func ensureSysProcAttr(sysProcAttr *syscall.SysProcAttr) (*syscall.SysProcAttr, error) {
	// NOTE: Cannot setsid and and setpgid in one child. Would need double fork or exec,
	// which makes things very hard.
	if sysProcAttr != nil && sysProcAttr.Setsid {
		return nil, fmt.Errorf("May not be used with a SysProcAttr.Setsid = true")
	}

	if sysProcAttr == nil {
		sysProcAttr = &syscall.SysProcAttr{}
	}

	sysProcAttr.Setpgid = true
	return sysProcAttr, nil
}

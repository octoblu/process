package process

import "syscall"

func ensureSysProcAttr(sysProcAttr *syscall.SysProcAttr) (*syscall.SysProcAttr, error) {
	if sysProcAttr != nil {
		return sysProcAttr, nil
	}

	return &syscall.SysProcAttr{}, nil
}

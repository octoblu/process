package process

import "syscall"

// softSignal returns this operating system's expected
// quit signal
func softSignal() syscall.Signal {
	return syscall.SIGTERM
}

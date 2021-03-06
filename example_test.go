package process_test

import (
	"os/exec"
	"time"

	"github.com/octoblu/process"
)

func ExampleBackground() {
	cmd := exec.Command("true")
	group, err := process.Background(cmd)
	if err != nil {
		panic(err)
	}
	group.Terminate(1 * time.Second)
}

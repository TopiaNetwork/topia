package block

import (
	"runtime"
	"fmt"
	"launchpad.net/gommap"
)

func system_entrance(){
	sysType := runtime.GOOS

	if sysType == 'linux' {
		// LINUX系统
		fmt.Println("Linux system")
	}

	if sysType == 'windows' {
		// windows系统
		fmt.Println("win system")
	}
}

package main

import (
	"fmt"
	"os"
	"os/exec"
	"path"
	"runtime"
)

func main() {
	if runtime.GOOS == "linux" {
		userPath, _ := os.Getwd()
		var abPath string
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			abPath = path.Dir(filename)
		} else {
			fmt.Println("get file path err")
			return
		}
		err := os.Chdir(abPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}

		fmt.Println("Module test with CGO is running, please wait...")
		output, err := exec.Command("./ed25519_cgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		fmt.Println(string(output))

		fmt.Println("Module test without CGO is running, please wait...")
		output, err = exec.Command("./ed25519_noncgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		fmt.Println(string(output))
		err = os.Chdir(userPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}
	} else if runtime.GOOS == "darwin" {
		userPath, _ := os.Getwd()
		var abPath string
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			abPath = path.Dir(filename)
		} else {
			fmt.Println("get file path err")
			return
		}
		err := os.Chdir(abPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}

		fmt.Println("Module test with CGO is running, please wait...")
		output, err := exec.Command("chmod", "+x", "./ed25519_cgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		output, err = exec.Command("./ed25519_cgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		fmt.Println(string(output))

		fmt.Println("Module test without CGO is running, please wait...")
		output, err = exec.Command("chmod", "+x", "./ed25519_noncgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		output, err = exec.Command("./ed25519_noncgo_test_script.sh").Output()
		if err != nil {
			fmt.Println("run module test shell script failed:", err)
			return
		}
		fmt.Println(string(output))

		err = os.Chdir(userPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}
	} else if runtime.GOOS == "windows" {
		userPath, _ := os.Getwd()
		var abPath string
		_, filename, _, ok := runtime.Caller(0)
		if ok {
			abPath = path.Dir(filename)
		} else {
			fmt.Println("get file path err")
			return
		}
		err := os.Chdir(abPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}

		fmt.Println("Module test with CGO is running, please wait...")
		cmd := exec.Command("powershell", ".\\ed25519_cgo_test_script.bat")
		output, err := cmd.CombinedOutput()
		if err != nil {
			fmt.Println("run .bat failed:", err)
			return
		}
		fmt.Println(string(output))

		fmt.Println("Module test without CGO is running, please wait...")
		cmd = exec.Command("powershell", ".\\ed25519_noncgo_test_script.bat")
		output, err = cmd.CombinedOutput()
		if err != nil {
			fmt.Println("run .bat failed:", err)
			return
		}
		fmt.Println(string(output))
		err = os.Chdir(userPath)
		if err != nil {
			fmt.Println("change dir err:", err)
			return
		}
	} else {
		fmt.Println("Your OS is not supported yet.")
	}
}

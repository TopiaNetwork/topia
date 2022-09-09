//go:build windows

package file_key_store

import (
	"golang.org/x/sys/windows"
	"syscall"
	"unsafe"
)

type processStatus struct {
	ID        int
	IsRunning bool
}
type windowsProcess struct {
	ProcessID int
}

const th32CsSnapProcess = 0x00000002

func isPIDAlive(pID int) (bool, error) {
	status := processStatus{ID: pID}

	procs, err := processes()
	if err != nil {
		return false, err
	}

	process := findProcessByID(procs, pID)
	if process != nil {
		status.IsRunning = true
	}

	return status.IsRunning, nil

}

func processes() ([]windowsProcess, error) {
	handle, err := windows.CreateToolhelp32Snapshot(th32CsSnapProcess, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]windowsProcess, 0, 50)
	for {
		results = append(results, newWindowsProcess(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func newWindowsProcess(e *windows.ProcessEntry32) windowsProcess {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return windowsProcess{
		ProcessID: int(e.ProcessID),
	}
}

func findProcessByID(processes []windowsProcess, pID int) *windowsProcess {
	for _, p := range processes {
		if pID == p.ProcessID {
			return &p
		}
	}
	return nil
}

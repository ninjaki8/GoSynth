package main

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

func isPluggedInWindows() bool {
	cmd := exec.Command("powershell.exe", "-Command", `Get-PnpDevice -PresentOnly | Where-Object { $_.InstanceId -like 'USB\VID_*' } | Select-Object -ExpandProperty InstanceId`)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running PowerShell:", err)
		return false
	}

	return strings.Contains(string(output), USB_VENDOR_ID_WINDOWS)
}

func lsusbAvailable() bool {
	_, err := exec.LookPath("lsusb")
	return err == nil
}

func isPluggedInLinux() bool {
	if !lsusbAvailable() {
		fmt.Println("Error: 'lsusb' not found. Please install it (e.g., with 'sudo pacman -S usbutils').")
		return false
	}

	out, err := exec.Command("lsusb").Output()
	if err != nil {
		fmt.Println("Failed to run lsusb:", err)
		return false
	}

	lines := strings.Split(string(out), "\n")
	for _, line := range lines {
		if strings.Contains(line, USB_VENDOR_ID_LINUX) {
			return true
		}
	}

	return false
}

func isQuestUsbConnected() bool {
	switch runtime.GOOS {
	case "windows":
		if isPluggedInWindows() {
			return true
		}

	case "linux":
		if isPluggedInLinux() {
			return true
		}
	}

	return false
}

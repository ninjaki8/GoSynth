package main

import (
	"fmt"
	"os/exec"
	"strings"
)

func questIsPluggedInWin() bool {
	cmd := exec.Command("powershell", "-Command", `Get-PnpDevice -PresentOnly | Where-Object { $_.InstanceId -match '^USB' }`)
	output, err := cmd.Output()
	if err != nil {
		fmt.Println("Error running PowerShell:", err)
		return false
	}

	isFound := strings.Contains(string(output), "VID_2833")
	if isFound {
		fmt.Println("[USB CHECK] Quest 3 is plugged in via USB.")
		return true
	}

	fmt.Println("[USB CHECK] Quest 3 not found.")
	return false
}

func lsusbAvailable() bool {
	_, err := exec.LookPath("lsusb")
	return err == nil
}

func questIsPluggedInLinux() bool {
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
		if strings.Contains(line, "2833:") {
			fmt.Println("[USB CHECK] Quest 3 is plugged in via USB.")
			return true
		}
	}

	fmt.Println("[USB CHECK] Quest 3 not found.")
	return false
}

func checkMissingDevice(isAdbEmptyList bool) bool {

	os := getOperatingSystem()

	switch os {
	case "windows":
		isQuestPluggedIn := questIsPluggedInWin()
		if isQuestPluggedIn && isAdbEmptyList {
			return true
		}

	case "linux":
		isQuestPluggedIn := questIsPluggedInLinux()
		if isQuestPluggedIn && isAdbEmptyList {
			return true
		}
	}

	return false
}

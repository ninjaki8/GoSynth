package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// startAdbServer ensures the ADB server is running and prints its status.
func startAdbServer(adbPath string) error {
	cmd := exec.Command(adbPath, "start-server")
	err := cmd.Run()

	if err != nil {
		return fmt.Errorf("failed to start ADB server: %v", err)
	}

	return nil
}

func getAdbVersion(adbPath string) {
	cmd := exec.Command(adbPath, "version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("[ERROR] Failed to get adb version: %v\n", err)
		return
	}

	fmt.Printf("%s\n", string(output))
}

func IsAdbEmptyDeviceList(adbPath string) (bool, error) {
	cmd := exec.Command(adbPath, "devices", "-l")
	output, err := cmd.Output()
	if err != nil {
		return false, err
	}

	// Check for empty device list
	outputStr := strings.TrimSpace(string(output))
	if outputStr == "List of devices attached" {
		return true, nil
	}

	return false, nil
}

// getDeviceSerial scans all connected devices and extracts the serial of the matching device model.
func getDeviceSerial(adbPath string, deviceModel string) (string, error) {
	cmd := exec.Command(adbPath, "devices", "-l")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}

	scanner := bufio.NewScanner(bytes.NewReader(output))
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "List of devices") || strings.TrimSpace(line) == "" {
			continue
		}

		fields := strings.Fields(line)
		if len(fields) < 2 || fields[1] != "device" {
			continue
		}

		serial := fields[0]
		for _, field := range fields {
			if strings.HasPrefix(field, "model:") {
				model := strings.TrimPrefix(field, "model:")
				if model == deviceModel {
					fmt.Printf("Model %s with serial %s found\n", model, serial)
					return serial, nil
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("%s not found", deviceModel)
}

// adbRemoteDirExists checks whether a directory exists on the device at the given path.
func adbRemoteDirExists(adbPath string, serial, remotePath string) error {
	cmd := exec.Command(adbPath, "-s", serial, "shell", "ls", remotePath)

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err := cmd.Run()

	// If no error, directory exists
	if err == nil {
		fmt.Println("[OK] Custom songs folder exists")
		return nil
	}

	// Check if error output is "No such file or directory"
	if strings.Contains(stderr.String(), "No such file or directory") {
		return fmt.Errorf("%v does not exist", remotePath)
	}

	// Some other error
	return fmt.Errorf("failed checking directory: %v\nstderr: %s", err, stderr.String())
}

// getDeviceSynthFiles creates a map of file names from the contents of a specified folder on the connected device.
func getDeviceSynthFiles(adbPath string, folderPath string, serial string) map[string]bool {
	cmd := exec.Command(adbPath, "-s", serial, "shell", "ls", folderPath)

	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Error listing folder %s: %v\n", folderPath, err)
		return map[string]bool{}
	}

	lines := strings.Split(string(output), "\n")

	synthFileNamesMap := make(map[string]bool)
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			synthFileNamesMap[trimmed] = true
		}
	}

	// Return the map of file names
	return synthFileNamesMap
}

func pushBeatmap(adbPath string, filePath string, deviceSerial string, remoteDir string) error {
	if deviceSerial == "" {
		return fmt.Errorf("device serial empty")
	}

	// Push to device
	cmd := exec.Command(adbPath, "-s", deviceSerial, "push", filePath, remoteDir)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("adb push failed: %v\nOutput: %s", err, string(output))
	}

	// Clean up
	err = os.Remove(filePath)
	if err != nil {
		fmt.Printf("[WARNING] Failed to delete temp file %s: %v\n", filePath, err)
	}

	return nil
}

package networkmanager

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// readEnvFile reads environment variables from a file
func readEnvFile(filename string) (map[string]string, error) {
	envVars := make(map[string]string)

	file, err := os.Open(filename)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Handle export statements
		if after, ok := strings.CutPrefix(line, "export "); ok {
			line = after
		}

		// Parse key=value
		if pair := strings.SplitN(line, "=", 2); len(pair) == 2 {
			key := strings.TrimSpace(pair[0])
			value := strings.Trim(strings.TrimSpace(pair[1]), `"'`)
			envVars[key] = value
		}
	}

	return envVars, scanner.Err()
}

// setSystemEnv sets an environment variable system-wide on the Pi
func setSystemEnv(key, value string) error {
	envFile := "/etc/environment"

	// Read existing content
	var lines []string
	if file, err := os.Open(envFile); err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			lines = append(lines, scanner.Text())
		}
		file.Close()
	}

	// Check if key already exists and update it
	keyFound := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = fmt.Sprintf("%s=\"%s\"", key, value)
			keyFound = true
			break
		}
	}

	// If key not found, append it
	if !keyFound {
		lines = append(lines, fmt.Sprintf("%s=\"%s\"", key, value))
	}

	// Write back to file
	file, err := os.OpenFile(envFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %v", envFile, err)
	}
	defer file.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(file, line); err != nil {
			return fmt.Errorf("failed to write to %s: %v", envFile, err)
		}
	}

	// Set the environment variable for the current process immediately
	os.Setenv(key, value)

	return nil
}

// removeSystemEnv removes an environment variable from system-wide configuration
func removeSystemEnv(key string) error {
	envFile := "/etc/environment"

	file, err := os.Open(envFile)
	if err != nil {
		// File doesn't exist, nothing to remove
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer file.Close()

	var lines []string
	var modified bool

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if strings.HasPrefix(trimmed, key+"=") {
			modified = true
			continue // Skip this line
		}

		lines = append(lines, line)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	// Only rewrite file if we modified it
	if !modified {
		return nil
	}

	// Write back to file
	outFile, err := os.OpenFile(envFile, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("failed to open %s for writing: %v", envFile, err)
	}
	defer outFile.Close()

	for _, line := range lines {
		if _, err := fmt.Fprintln(outFile, line); err != nil {
			return fmt.Errorf("failed to write to %s: %v", envFile, err)
		}
	}

	// Unset the environment variable for the current process
	os.Unsetenv(key)

	return nil
}

// reloadSystemEnv attempts to reload environment variables system-wide
func reloadSystemEnv() error {
	// Source the environment file for systemd services
	cmd := exec.Command("systemctl", "daemon-reload")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to reload systemd: %v", err)
	}

	return nil
}

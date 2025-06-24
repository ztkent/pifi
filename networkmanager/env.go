package networkmanager

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
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

const (
	managedEnvFile = "/etc/default/pifi_managed_vars"
)

// ManagedEnvVars represents the list of environment variables managed by the service
type ManagedEnvVars struct {
	Variables []string `json:"variables"`
}

// readManagedEnvList reads the list of managed environment variables
func readManagedEnvList() ([]string, error) {
	// Try system location first
	if vars, err := readManagedEnvFile(managedEnvFile); err == nil {
		return vars, nil
	}

	// Try user location
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return []string{}, nil // Return empty list if can't determine home
	}

	userManagedFile := filepath.Join(homeDir, ".pifi_managed_vars")
	if vars, err := readManagedEnvFile(userManagedFile); err == nil {
		return vars, nil
	}

	// Return empty list if no file found
	return []string{}, nil
}

// writeManagedEnvList writes the list of managed environment variables
func writeManagedEnvList(variables []string) error {
	managed := ManagedEnvVars{Variables: variables}
	data, err := json.Marshal(managed)
	if err != nil {
		return err
	}

	// Try system location first
	if err := writeManagedEnvFile(managedEnvFile, data); err != nil {
		// Fallback to user location
		homeDir, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return fmt.Errorf("failed to write managed vars list: no write access to system files and cannot determine home directory")
		}

		userManagedFile := filepath.Join(homeDir, ".pifi_managed_vars")
		if err := writeManagedEnvFile(userManagedFile, data); err != nil {
			return fmt.Errorf("failed to write managed vars list: %v", err)
		}
	}

	return nil
}

// addToManagedList adds a variable to the managed list
func addToManagedList(key string) error {
	variables, err := readManagedEnvList()
	if err != nil {
		return err
	}

	// Check if already in list
	for _, v := range variables {
		if v == key {
			return nil // Already managed
		}
	}

	// Add to list
	variables = append(variables, key)
	return writeManagedEnvList(variables)
}

// removeFromManagedList removes a variable from the managed list
func removeFromManagedList(key string) error {
	variables, err := readManagedEnvList()
	if err != nil {
		return err
	}

	// Remove from list
	newVariables := make([]string, 0, len(variables))
	for _, v := range variables {
		if v != key {
			newVariables = append(newVariables, v)
		}
	}

	return writeManagedEnvList(newVariables)
}

// readManagedEnvFile reads managed variables from a JSON file
func readManagedEnvFile(filename string) ([]string, error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return nil, err
	}

	var managed ManagedEnvVars
	if err := json.Unmarshal(data, &managed); err != nil {
		return nil, err
	}

	return managed.Variables, nil
}

// writeManagedEnvFile writes managed variables to a JSON file
func writeManagedEnvFile(filename string, data []byte) error {
	return os.WriteFile(filename, data, 0644)
}

// getManagedEnvironmentVariables returns only the environment variables that are managed by the service
func getManagedEnvironmentVariables() (map[string]string, error) {
	// Get the list of managed variables
	managedVars, err := readManagedEnvList()
	if err != nil {
		return nil, err
	}

	if len(managedVars) == 0 {
		return map[string]string{}, nil
	}

	// Get all environment variables
	allEnvVars := make(map[string]string)

	// Try to read from common environment sources
	sources := []string{
		"/etc/environment",
		"/etc/default/pifi",
		"/home/pi/.bashrc",
	}

	for _, source := range sources {
		if vars, err := readEnvFile(source); err == nil {
			for k, v := range vars {
				allEnvVars[k] = v
			}
		}
	}

	// Add current process environment
	for _, env := range os.Environ() {
		if pair := strings.SplitN(env, "=", 2); len(pair) == 2 {
			allEnvVars[pair[0]] = pair[1]
		}
	}

	// Filter to only include managed variables
	managedEnvVars := make(map[string]string)
	for _, key := range managedVars {
		if value, exists := allEnvVars[key]; exists {
			managedEnvVars[key] = value
		}
	}

	return managedEnvVars, nil
}

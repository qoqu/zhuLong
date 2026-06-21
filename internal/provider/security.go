package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// KeySource represents the source of an API key
type KeySource string

const (
	KeySourceEnv      KeySource = "env"      // Environment variable
	KeySourceFile     KeySource = "file"     // File
	KeySourceKeychain KeySource = "keychain" // OS keychain
	KeySourcePrompt   KeySource = "prompt"   // User prompt
)

// SecurityConfig contains configuration for API key security
type SecurityConfig struct {
	// KeyEnvVar is the environment variable name for the API key
	KeyEnvVar string

	// KeyFile is the file path for the API key
	KeyFile string

	// KeychainService is the keychain service name
	KeychainService string

	// AllowPrompt allows prompting for the key
	AllowPrompt bool

	// MaskKey masks the key in logs
	MaskKey bool
}

// DefaultSecurityConfig returns default security configuration
func DefaultSecurityConfig() *SecurityConfig {
	return &SecurityConfig{
		KeyEnvVar:       "DEEPSEEK_API_KEY",
		KeyFile:         "",
		KeychainService: "zhulong",
		AllowPrompt:     false,
		MaskKey:         true,
	}
}

// KeyManager manages API keys securely
type KeyManager struct {
	config *SecurityConfig
}

// NewKeyManager creates a new key manager
func NewKeyManager(config *SecurityConfig) *KeyManager {
	if config == nil {
		config = DefaultSecurityConfig()
	}

	return &KeyManager{
		config: config,
	}
}

// GetKey gets the API key from the configured source
func (km *KeyManager) GetKey() (string, KeySource, error) {
	// Try environment variable first
	if km.config.KeyEnvVar != "" {
		key := os.Getenv(km.config.KeyEnvVar)
		if key != "" {
			return key, KeySourceEnv, nil
		}
	}

	// Try file
	if km.config.KeyFile != "" {
		key, err := km.getKeyFromFile(km.config.KeyFile)
		if err == nil && key != "" {
			return key, KeySourceFile, nil
		}
	}

	// Try default file locations
	key, err := km.getKeyFromDefaultFile()
	if err == nil && key != "" {
		return key, KeySourceFile, nil
	}

	// Try OS keychain (platform-specific)
	key, err = km.getKeyFromKeychain()
	if err == nil && key != "" {
		return key, KeySourceKeychain, nil
	}

	return "", "", fmt.Errorf("no API key found")
}

// SetKey sets the API key
func (km *KeyManager) SetKey(key string, source KeySource) error {
	switch source {
	case KeySourceEnv:
		return os.Setenv(km.config.KeyEnvVar, key)
	case KeySourceFile:
		return km.setKeyToFile(km.config.KeyFile, key)
	case KeySourceKeychain:
		return km.setKeyToKeychain(key)
	default:
		return fmt.Errorf("unsupported key source: %s", source)
	}
}

// MaskKey masks the API key for display
func (km *KeyManager) MaskKey(key string) string {
	if !km.config.MaskKey {
		return key
	}

	if len(key) <= 8 {
		return "****"
	}

	return key[:4] + "****" + key[len(key)-4:]
}

// getKeyFromFile gets the key from a file
func (km *KeyManager) getKeyFromFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	key := strings.TrimSpace(string(content))
	return key, nil
}

// setKeyToFile sets the key to a file
func (km *KeyManager) setKeyToFile(path string, key string) error {
	// Create directory if needed
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Write key to file with restrictive permissions
	if err := os.WriteFile(path, []byte(key), 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	return nil
}

// getKeyFromDefaultFile gets the key from default file locations
func (km *KeyManager) getKeyFromDefaultFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	// Try ~/.zhulong/api_key
	keyFile := filepath.Join(home, ".zhulong", "api_key")
	key, err := km.getKeyFromFile(keyFile)
	if err == nil && key != "" {
		return key, nil
	}

	// Try ~/.config/zhulong/api_key
	keyFile = filepath.Join(home, ".config", "zhulong", "api_key")
	key, err = km.getKeyFromFile(keyFile)
	if err == nil && key != "" {
		return key, nil
	}

	return "", fmt.Errorf("no default key file found")
}

// getKeyFromKeychain gets the key from OS keychain
func (km *KeyManager) getKeyFromKeychain() (string, error) {
	// Platform-specific implementation
	switch runtime.GOOS {
	case "darwin":
		return km.getKeyFromMacOSKeychain()
	case "windows":
		return km.getKeyFromWindowsCredentialManager()
	case "linux":
		return km.getKeyFromLinuxSecretService()
	default:
		return "", fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// setKeyToKeychain sets the key to OS keychain
func (km *KeyManager) setKeyToKeychain(key string) error {
	// Platform-specific implementation
	switch runtime.GOOS {
	case "darwin":
		return km.setKeyToMacOSKeychain(key)
	case "windows":
		return km.setKeyToWindowsCredentialManager(key)
	case "linux":
		return km.setKeyToLinuxSecretService(key)
	default:
		return fmt.Errorf("unsupported platform: %s", runtime.GOOS)
	}
}

// macOS Keychain implementation
func (km *KeyManager) getKeyFromMacOSKeychain() (string, error) {
	// In a real implementation, this would use the macOS Keychain API
	// For now, return empty
	return "", fmt.Errorf("macOS keychain not implemented")
}

func (km *KeyManager) setKeyToMacOSKeychain(key string) error {
	// In a real implementation, this would use the macOS Keychain API
	return fmt.Errorf("macOS keychain not implemented")
}

// Windows Credential Manager implementation
func (km *KeyManager) getKeyFromWindowsCredentialManager() (string, error) {
	// In a real implementation, this would use Windows Credential Manager
	// For now, return empty
	return "", fmt.Errorf("Windows Credential Manager not implemented")
}

func (km *KeyManager) setKeyToWindowsCredentialManager(key string) error {
	// In a real implementation, this would use Windows Credential Manager
	return fmt.Errorf("Windows Credential Manager not implemented")
}

// Linux Secret Service implementation
func (km *KeyManager) getKeyFromLinuxSecretService() (string, error) {
	// In a real implementation, this would use libsecret
	// For now, return empty
	return "", fmt.Errorf("Linux Secret Service not implemented")
}

func (km *KeyManager) setKeyToLinuxSecretService(key string) error {
	// In a real implementation, this would use libsecret
	return fmt.Errorf("Linux Secret Service not implemented")
}

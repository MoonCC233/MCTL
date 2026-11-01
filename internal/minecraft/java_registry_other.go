//go:build !windows

package minecraft

// windowsRegistryJavaHomes is a no-op outside Windows.
func windowsRegistryJavaHomes() []string { return nil }

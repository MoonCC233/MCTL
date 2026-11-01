//go:build windows

package minecraft

import "golang.org/x/sys/windows/registry"

// windowsRegistryJavaRoots are the registry keys vendors use to record the
// install directory of a JDK or JRE.
var windowsRegistryJavaRoots = []string{
	`SOFTWARE\JavaSoft\JDK`,
	`SOFTWARE\JavaSoft\Java Development Kit`,
	`SOFTWARE\JavaSoft\JRE`,
	`SOFTWARE\JavaSoft\Java Runtime Environment`,
	`SOFTWARE\Eclipse Adoptium\JDK`,
	`SOFTWARE\Eclipse Foundation\JDK`,
	`SOFTWARE\Microsoft\JDK`,
	`SOFTWARE\Azul Systems\Zulu`,
	`SOFTWARE\Amazon Corretto`,
	`SOFTWARE\BellSoft\LibericaJDK`,
	`SOFTWARE\Semeru\JDK`,
	`SOFTWARE\AdoptOpenJDK\JDK`,
	`SOFTWARE\WOW6432Node\JavaSoft\JDK`,
	`SOFTWARE\WOW6432Node\JavaSoft\Java Development Kit`,
	`SOFTWARE\WOW6432Node\JavaSoft\JRE`,
	`SOFTWARE\WOW6432Node\Eclipse Adoptium\JDK`,
	`SOFTWARE\WOW6432Node\Microsoft\JDK`,
}

var windowsRegistryJavaValues = []string{"JavaHome", "Path", "InstallationPath", "InstallPath"}

// windowsRegistryJavaHomes reads vendor install locations from the registry.
func windowsRegistryJavaHomes() []string {
	var homes []string
	for _, root := range windowsRegistryJavaRoots {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, root, registry.READ|registry.WOW64_64KEY)
		if err != nil {
			continue
		}
		names, err := key.ReadSubKeyNames(-1)
		key.Close()
		if err != nil {
			continue
		}
		for _, name := range names {
			sub, err := registry.OpenKey(registry.LOCAL_MACHINE, root+`\`+name, registry.READ|registry.WOW64_64KEY)
			if err != nil {
				continue
			}
			for _, value := range windowsRegistryJavaValues {
				if v, _, err := sub.GetStringValue(value); err == nil && v != "" {
					homes = append(homes, v)
				}
			}
			sub.Close()
		}
	}
	return homes
}

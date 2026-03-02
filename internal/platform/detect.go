package platform

import "runtime"

// Platform represents the detected operating system.
type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformLinux   Platform = "linux"
	PlatformWindows Platform = "windows"
	PlatformUnknown Platform = "unknown"
)

// Detect returns the current platform based on runtime.GOOS.
func Detect() Platform {
	switch runtime.GOOS {
	case "darwin":
		return PlatformMacOS
	case "linux":
		return PlatformLinux
	case "windows":
		return PlatformWindows
	default:
		return PlatformUnknown
	}
}

// ServiceName is the identifier used for OS-native service registration.
const ServiceName = "com.kronos.agent"

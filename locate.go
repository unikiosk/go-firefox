package gofirefox

import (
	"os"
	"runtime"
)

// FirefoxExecutable returns a string which points to the preferred Firefox
// executable file.
var FirefoxExecutable = LocateFirefox

// LocateFirefox returns a path to the Firefox binary, or an empty string if
// Firefox installation is not found.
func LocateFirefox() string {

	// If env variable "GOFIREFOX_BIN" specified and it exists
	if path, ok := os.LookupEnv("GOFIREFOX_BIN"); ok {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	// https://kb.mozillazine.org/Installation_directory

	var paths []string
	switch runtime.GOOS {
	case "darwin":
		/*
			Firefox	/Applications/Firefox.app
			Thunderbird	/Applications/Thunderbird.app
			Mozilla Suite	/Applications/Mozilla.app
		*/
		paths = []string{
			"/Applications/Firefox.app",
			"/Applications/Thunderbird.app",
			"/Applications/Mozilla.app",
			"/usr/bin/firefox",
		}
	case "windows":
		/*
			Firefox	C:\Program Files\Mozilla Firefox\
			Firefox (64-bit Windows)	C:\Program Files (x86)\Mozilla Firefox\
			Thunderbird	C:\Program Files\Mozilla Thunderbird\
			Mozilla Suite	C:\Program Files\mozilla.org\Mozilla\
			SeaMonkey 1.x	C:\Program Files\mozilla.org\SeaMonkey\
			SeaMonkey 2.0	C:\Program Files\SeaMonkey\
		*/
		paths = []string{
			os.Getenv("ProgramFiles") + "/Mozilla Firefox/firefox.exe",
			os.Getenv("ProgramFiles(x86)") + "/Mozilla Firefox/firefox.exe",
			os.Getenv("ProgramFiles") + "/Mozilla Thunderbird/thunderbird.exe",
			os.Getenv("ProgramFiles") + "/mozilla.org/Mozilla/firefox.exe",
			os.Getenv("ProgramFiles") + "/mozilla.org/SeaMonkey/firefox.exe",
			os.Getenv("ProgramFiles") + "/SeaMonkey/firefox.exe",
		}
	default:
		// TODO: extend more path if we need to
		paths = []string{
			"/usr/bin/firefox",
		}
	}

	for _, path := range paths {
		if _, err := os.Stat(path); os.IsNotExist(err) {
			continue
		}
		return path
	}
	return ""
}

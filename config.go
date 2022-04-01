package gofirefox

import (
	"os"
)

type Config struct {
	// ProfileDir is directory where semi-persistent profile is stored
	ProfileDir string
	// FirefoxBin is override location for firefox binary
	FirefoxBin string
	// ProfileLocationURL is profile location file
	ProfileLocationURL string
}

const (
	PreferenceFile         = "prefs.js"
	DefaultProfileLocation = "https://raw.githubusercontent.com/unikiosk/user.js/master/user.js"
)

func getConfig() (*Config, error) {
	c := Config{}

	if path, ok := os.LookupEnv("GOFIREFOX_BIN"); ok {
		if _, err := os.Stat(path); err == nil {
			c.FirefoxBin = path
		}
	} else {
		c.FirefoxBin = FirefoxExecutable()
	}

	if profileDir, ok := os.LookupEnv("GOFIREFOX_PROFILE_DIR"); ok {
		err := os.MkdirAll(profileDir, os.ModeDir)
		if err != nil {
			return nil, err
		}
		c.ProfileDir = profileDir
	} else {
		tempDir, err := os.MkdirTemp(os.TempDir(), "gofirefox")
		if err != nil {
			return nil, err
		}
		c.ProfileDir = tempDir
	}

	if profileLocation, ok := os.LookupEnv("GOFIREFOX_PROFILE_LOCATION"); ok {
		c.ProfileLocationURL = profileLocation
	} else {
		c.ProfileLocationURL = DefaultProfileLocation
	}

	return &c, nil
}

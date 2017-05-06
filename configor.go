package configor

import (
	"os"
	"regexp"
)

type Configor struct {
	*Config
}

type Config struct {
	Environment string
	ENVPrefix   string
}

// New initialize a Configor
func New(config *Config) *Configor {
	if config == nil {
		config = &Config{}
	}
	return &Configor{Config: config}
}

func isTestEnv() (returnStr string) {
	isTest, _ := regexp.MatchString("/_test/", os.Args[0])

	if isTest {
		returnStr = "test"
	} else {
		returnStr = ""
	}
	return
}

// GetEnvironment returns configor.Environment str
func (configor *Configor) GetEnvironment() string {
	if configor.Environment != "" {
		return configor.Environment
	}
	isTestStr := isTestEnv()
	envStrings := []string{os.Getenv("CONFIGOR_ENV"), isTestStr, "development"}

	for _, envStr := range envStrings {
		if envStr != "" {
			configor.Environment = envStr
			break
		}
	}
	return configor.Environment
}

// Load will unmarshal configurations to struct from files that you provide
func (configor *Configor) Load(config interface{}, files ...string) (err error) {
	for _, file := range configor.getConfigurationFiles(files...) {
		err = processFile(config, file)

		if err != nil {
			return
		}
	}
	prefix := configor.getENVPrefix(config)

	if prefix == "-" {
		err = processTags(config)
	} else {
		err = processTags(config, prefix)
	}
	return
}

// ENV return environment
func ENV() string {
	return New(nil).GetEnvironment()
}

// Load will unmarshal configurations to struct from files that you provide
func Load(config interface{}, files ...string) error {
	return New(nil).Load(config, files...)
}

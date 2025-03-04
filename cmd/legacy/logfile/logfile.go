package logfile

import (
	"fmt"
	"path/filepath"

	"github.com/wakatime/wakatime-cli/pkg/config"
	"github.com/wakatime/wakatime-cli/pkg/vipertools"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/viper"
)

const defaultFile = ".wakatime.log"

// Params contains log file parameters.
type Params struct {
	File     string
	ToStdout bool
	Verbose  bool
}

// LoadParams loads needed data from the configuration file.
func LoadParams(v *viper.Viper) (Params, error) {
	var debug bool
	if b := v.GetBool("settings.debug"); v.IsSet("settings.debug") {
		debug = b
	}

	logFile, ok := vipertools.FirstNonEmptyString(v, "log-file", "logfile", "settings.log_file")
	if ok {
		p, err := homedir.Expand(logFile)
		if err != nil {
			return Params{},
				ErrLogFile(fmt.Sprintf("failed expanding log file: %s", err))
		}

		return Params{
			File:    p,
			Verbose: v.GetBool("verbose") || debug,
		}, nil
	}

	home, err := config.WakaHomeDir()
	if err != nil {
		return Params{}, fmt.Errorf("failed getting user's home directory: %s", err)
	}

	return Params{
		File:     filepath.Join(home, defaultFile),
		ToStdout: v.GetBool("log-to-stdout"),
		Verbose:  v.GetBool("verbose") || debug,
	}, nil
}

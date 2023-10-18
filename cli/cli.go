package cli

import (
	"github.com/F1997/nightingale/cli/upgrade"
)

func Upgrade(configFile string) error {
	return upgrade.Upgrade(configFile)
}

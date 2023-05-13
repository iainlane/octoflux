/*
Copyright © 2022 Iain Lane <iain@orangesquash.org.uk>

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

You should have received a copy of the GNU General Public License
along with this program. If not, see <http://www.gnu.org/licenses/>.
*/
package octopus

import (
	"context"
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/adrg/xdg"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/iainlane/octoflux/internal/conf"
)

const (
	envPrefix = "OCTOPUS"
)

var config conf.Conf
var cfgFile string

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "octopus",
	Short: "A CLI for Octopus Energy",
	Long: `Octopus is a CLI for Octopus Energy. It allows you to view tariffs,
see your current and historical usage, and export the same as Prometheus
metrics.`,
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return preRun(cmd.Context())
	},
}

func preRun(ctx context.Context) error {
	if config.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debugln("Debug logging enabled")
	}

	return nil
}

func Execute() {
	ctx, cancel := context.WithCancel(context.Background())
	err := rootCmd.ExecuteContext(ctx)
	go func() {
		// exit 0 when context is done (sigint or sigterm)
		<-ctx.Done()
		cancel()
		os.Exit(0)
	}()
	defer cancel()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	cobra.OnInitialize(initConfig)

	rootCmd.Args = cobra.NoArgs

	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", fmt.Sprintf("config file (default is %s", path.Join(xdg.ConfigHome, "octopus", "octopus.yaml")))
	rootCmd.PersistentFlags().BoolVar(&config.Debug, "debug", false, "enable debug logging")

	rootCmd.PersistentFlags().StringVarP(&config.OctopusAPIKey, "api-key", "k", "", "Octopus API key")
	rootCmd.MarkPersistentFlagRequired("api-key")

	rootCmd.PersistentFlags().StringVar(&config.ElectricityMPN, "electricity-mpn", "", "MPN for electricity")
	rootCmd.PersistentFlags().StringVar(&config.ElectricitySerial, "electricity-serial", "", "Serial for electricity")
	rootCmd.MarkFlagsRequiredTogether("electricity-mpn", "electricity-serial")

	rootCmd.PersistentFlags().StringVar(&config.GasMPN, "gas-mpn", "", "MPN for gas")
	rootCmd.PersistentFlags().StringVar(&config.GasSerial, "gas-serial", "", "Serial for gas")
	rootCmd.MarkFlagsRequiredTogether("gas-mpn", "gas-serial")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	v := viper.New()

	v.SetEnvPrefix(envPrefix)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.AddConfigPath(path.Join(xdg.ConfigHome, "octopus"))
		v.SetConfigType("yaml")
		v.SetConfigName("octopus")
	}

	v.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := v.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file: %s", err)
	}

	bindFlags(rootCmd, v)
}

func bindFlags(cmd *cobra.Command, v *viper.Viper) {
	cmd.Flags().VisitAll(func(f *pflag.Flag) {
		// Environment variables can't have dashes in them, so bind them to their equivalent
		// keys with underscores, e.g. --electricity-mpn becomes OCTOPUS_ELECTRICITY_MPN
		if strings.Contains(f.Name, "-") {
			envVarSuffix := strings.ToUpper(strings.ReplaceAll(f.Name, "-", "_"))
			v.BindEnv(f.Name, fmt.Sprintf("%s_%s", envPrefix, envVarSuffix))
		}

		// Apply the viper config value to the flag when the flag is not set and viper has a value
		if !f.Changed && v.IsSet(f.Name) {
			val := v.Get(f.Name)
			cmd.Flags().Set(f.Name, fmt.Sprintf("%v", val))
		}
	})
}

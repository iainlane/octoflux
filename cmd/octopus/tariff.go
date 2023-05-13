/*
Copyright © 2023 Iain Lane <iain@orangesquash.org.uk>

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
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/iainlane/octoflux/internal/octopus"
)

// tariffCmd represents the tariff command
var tariffCmd = &cobra.Command{
	Use:   "tariff",
	Short: "Show tariff details",
	Long:  `Show tariff details.`,
	Run: func(cmd *cobra.Command, args []string) {
		octopusClient := octopus.New(cmd.Context(), config)
		log.Debug("getting tariff details")
		octopusClient.
	},
}

func init() {
	rootCmd.AddCommand(tariffCmd)
}

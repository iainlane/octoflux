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
	"os"

	"github.com/danopstech/octopusenergy"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"

	"github.com/iainlane/octoflux/internal/octopus"
)

// usageCmd represents the usage command
var usageCmd = &cobra.Command{
	Use:   "usage",
	Short: "Print energy usage",
	Long:  `Print energy usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		octopusClient := octopus.New(cmd.Context(), config)
		totalElectricity, electricityGrouped, err := octopusClient.GetAllConsumption(octopusenergy.FuelTypeElectricity, "day")
		if err != nil {
			log.Errorf("error getting electricity consumption: %v", err)
			os.Exit(1)
		}
		// get last 30 days of electricity consumption
		var recentElectricity []float64
		var maxElectricity float64
		for _, consumption := range electricityGrouped[len(electricityGrouped)-30:] {
			if consumption.Consumption > maxElectricity {
				maxElectricity = consumption.Consumption
			}
			recentElectricity = append(recentElectricity, consumption.Consumption)
		}

		totalGas, gasGrouped, err := octopusClient.GetAllConsumption(octopusenergy.FuelTypeGas, "day")
		if err != nil {
			log.Errorf("error getting gas consumption: %v", err)
			os.Exit(1)
		}
		var recentGas []float64
		var maxGas float64
		// get last 30 days of gas consumption
		for _, consumption := range gasGrouped[len(gasGrouped)-30:] {
			if consumption.Consumption > maxGas {
				maxGas = consumption.Consumption
			}
			recentGas = append(recentGas, consumption.Consumption)
		}

		log.Printf("Total electricity: %.2f kWh", totalElectricity)
		log.Printf("Total gas: %.2f kWh", totalGas)
		log.Printf("Recent electricity: %.2f kWh", recentElectricity)
		log.Printf("Recent gas: %.2f kWh", recentGas)
	},
}

func init() {
	rootCmd.AddCommand(usageCmd)
}

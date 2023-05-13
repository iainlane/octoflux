/* octoflux
 * Copyright (C) 2021  Iain Lane
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/danopstech/octopusenergy"
	"github.com/iainlane/octoflux/internal/conf"
	"github.com/iainlane/octoflux/internal/octopus"

	ui "github.com/gizak/termui/v3"
	"github.com/gizak/termui/v3/widgets"
	"github.com/jessevdk/go-flags"

	log "github.com/sirupsen/logrus"
)

func main() {
	var conf conf.Conf

	parser := flags.NewParser(&conf, flags.Default)
	if _, err := parser.Parse(); err != nil {
		if flags.WroteHelp(err) {
			os.Exit(1)
		}
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	electricityOK := conf.ElectricityMPN != "" && conf.ElectricitySerial != ""
	gasOK := conf.GasMPN != "" && conf.GasSerial != ""

	if !electricityOK && !gasOK {
		os.Stderr.WriteString("must specify either electricity or gas MPN and serial\n")
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if (conf.ElectricityMPN == "" && conf.ElectricitySerial != "") || (conf.ElectricityMPN != "" && conf.ElectricitySerial == "") {
		os.Stderr.WriteString("must specify both electricity MPN and serial\n")
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if (conf.GasMPN == "" && conf.GasSerial != "") || (conf.GasMPN != "" && conf.GasSerial == "") {
		os.Stderr.WriteString("must specify both gas MPN and serial\n")
		parser.WriteHelp(os.Stderr)
		os.Exit(1)
	}

	if conf.Debug {
		log.SetLevel(log.DebugLevel)
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	go func() {
		// exit 0 when context is done (sigint or sigterm)
		<-ctx.Done()
		cancel()
		os.Exit(0)
	}()

	octopusClient := octopus.New(ctx, conf)

	_, electricityGrouped, err := octopusClient.GetAllConsumption(octopusenergy.FuelTypeElectricity, "day")
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

	_, gasGrouped, err := octopusClient.GetAllConsumption(octopusenergy.FuelTypeGas, "day")
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

	lc := widgets.NewPlot()
	lc.Title = "recent gas usage"
	lc.Data = make([][]float64, 1)
	lc.Data[0] = recentGas
	lc.SetRect(0, 0, 40, 20)
	lc.AxesColor = ui.ColorWhite
	lc.LineColors[0] = ui.ColorRed
	lc.Marker = widgets.MarkerDot
	lc.MaxVal = maxGas

	lc2 := widgets.NewPlot()
	lc2.Title = "recent electricity usage"
	lc2.Data = make([][]float64, 1)
	lc2.Data[0] = recentElectricity
	lc2.SetRect(40, 0, 80, 20)
	lc2.AxesColor = ui.ColorWhite
	lc2.LineColors[0] = ui.ColorRed
	lc2.Marker = widgets.MarkerDot
	lc2.MaxVal = maxElectricity

	if err := ui.Init(); err != nil {
		log.Fatalf("failed to initialize termui: %v", err)
	}
	defer ui.Close()

	ui.Render(lc, lc2)

	for e := range ui.PollEvents() {
		if e.Type == ui.KeyboardEvent {
			break
		}
	}
}

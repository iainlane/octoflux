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
	"sync"
	"time"

	"github.com/iainlane/octoflux/internal/conf"
	"github.com/iainlane/octoflux/internal/influx"
	"github.com/iainlane/octoflux/internal/octopus"

	"github.com/jessevdk/go-flags"
	"golang.org/x/sync/errgroup"

	log "github.com/sirupsen/logrus"
)

func getLastSubmission(ctx context.Context, c *conf.Conf, influxClient influx.Influx, fuelType string) (time.Time, error) {
	lastSubmission, err := influxClient.GetLastSubmission(ctx, c, fuelType)
	if err != nil {
		log.WithError(err).Errorf("Error getting last %s submission", fuelType)
		return time.Time{}, err
	}
	if lastSubmission.IsZero() {
		log.Infof("No previous %s submission found", fuelType)
	} else {
		log.Infof("Last %s submission in database: %s", fuelType, lastSubmission)
	}

	return lastSubmission, nil
}

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

	if conf.DryRun {
		log.Info("[dry-run] Running in dry-run mode, nothing will be written to Influx")
	}

	ctx, cancel := context.WithTimeout(context.Background(), time.Minute*10)
	defer cancel()
	grp, ctx := errgroup.WithContext(ctx)

	log.Debug("Creating influx client")
	influxClient, err := influx.MakeInfluxClient(ctx, &conf)
	if err != nil {
		log.WithError(err).Error("Error creating influx client")
		return
	}
	defer influxClient.Close()

	lastElectricitySubmission, err := getLastSubmission(ctx, &conf, influxClient, "electricity")
	if err != nil {
		os.Exit(1)
	}

	lastGasSubmission, err := getLastSubmission(ctx, &conf, influxClient, "gas")
	if err != nil {
		os.Exit(1)
	}

	var wg sync.WaitGroup
	c := make(chan *octopus.ConsumptionResponse, 10)

	octopus.GetConsumption(&wg, &conf, grp, ctx, &lastElectricitySubmission, &lastGasSubmission, c)
	grp.Go(func() error {
		wg.Wait()
		log.Info("Done getting consumption")
		close(c)
		return nil
	})

	grp.Go(func() error {
		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case val, more := <-c:
				if !more {
					return nil
				}
				err := influxClient.SubmitConsumption(ctx, &conf, val.FuelType.String(), val.Consumption, val.Period)
				if err != nil {
					log.WithError(err).Error("Error writing point")
					close(c)
					return err
				}
			}
		}
	})

	if err := grp.Wait(); err != nil {
		os.Exit(1)
	}
	log.Info("Done")
}

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

package octopus

import (
	"context"
	"net/http"
	"sync"
	"time"

	"github.com/danopstech/octopusenergy"
	"github.com/iainlane/octoflux/internal/conf"
	"golang.org/x/sync/errgroup"

	log "github.com/sirupsen/logrus"
)

type ConsumptionResponse struct {
	Consumption float64
	FuelType    octopusenergy.FuelType
	Period      time.Time
}

func get(ctx context.Context, client *octopusenergy.Client, opts octopusenergy.ConsumptionGetOptions, c chan<- *ConsumptionResponse) error {
	consumption, err := client.Consumption.GetPagesWithContext(ctx, &opts)

	if err != nil {
		log.WithFields(log.Fields{
			"fuel_type": opts.FuelType,
			"mpn":       opts.MPN,
			"serial":    opts.SerialNumber,
		}).WithError(err).Error("Error getting consumption")
		return err
	}

	if consumption.Results == nil {
		log.Infof("No new %s records to fetch", opts.FuelType)
		return nil
	}

	for _, cons := range consumption.Results {
		t, err := time.Parse(time.RFC3339, cons.IntervalStart)
		if err != nil {
			log.WithError(err).Errorf("Error parsing time: %s", cons.IntervalStart)
			return err
		}

		resp := &ConsumptionResponse{
			Consumption: cons.Consumption,
			FuelType:    opts.FuelType,
			Period:      t,
		}
		log.WithFields(log.Fields{
			"consumption": resp.Consumption,
			"fuel_type":   resp.FuelType,
			"period":      resp.Period,
		}).Debugf("Got consumption")
		c <- resp
	}

	return nil
}

func GetConsumption(wg *sync.WaitGroup, config *conf.Conf, grp *errgroup.Group, ctx context.Context, period *time.Time, c chan<- *ConsumptionResponse) {
	log.Debug("Creating octopus client")
	var netClient = http.Client{
		Timeout: time.Second * 10,
	}
	client := octopusenergy.NewClient(octopusenergy.NewConfig().
		WithApiKey(config.OctopusAPIKey).
		WithHTTPClient(netClient),
	)

	var orderByPeriod = "period"

	log.Debugf("Getting consumption since %s", period)
	if config.ElectricityMPN != "" {
		wg.Add(1)
	}
	if config.GasMPN != "" {
		wg.Add(1)
	}

	if config.ElectricityMPN != "" {
		grp.Go(func() error {
			defer wg.Done()
			opts := octopusenergy.ConsumptionGetOptions{
				MPN:          config.ElectricityMPN,
				SerialNumber: config.ElectricitySerial,
				FuelType:     octopusenergy.FuelTypeElectricity,
				PeriodFrom:   period,
				OrderBy:      &orderByPeriod,
			}
			return get(ctx, client, opts, c)
		})
	}

	if config.GasMPN != "" {
		grp.Go(func() error {
			defer wg.Done()
			opts := octopusenergy.ConsumptionGetOptions{
				MPN:          config.GasMPN,
				SerialNumber: config.GasSerial,
				FuelType:     octopusenergy.FuelTypeGas,
				PeriodFrom:   period,
				OrderBy:      &orderByPeriod,
			}
			return get(ctx, client, opts, c)
		})
	}
}

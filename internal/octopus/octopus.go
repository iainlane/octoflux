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

type OctopusClient struct {
	conf conf.Conf

	ctx    context.Context
	client *octopusenergy.Client
}

func New(ctx context.Context, conf conf.Conf) *OctopusClient {
	log.Debug("Creating octopus client")

	var netClient = http.Client{
		Timeout: time.Second * 10,
	}
	client := octopusenergy.NewClient(octopusenergy.NewConfig().
		WithApiKey(conf.OctopusAPIKey).
		WithHTTPClient(netClient),
	)
	log.Debug("Created octopus client")

	return &OctopusClient{
		conf: conf,

		ctx:    ctx,
		client: client,
	}
}

func (o *OctopusClient) getConsumption(opts octopusenergy.ConsumptionGetOptions, c chan<- *ConsumptionResponse) error {
	defer close(c)
	consumption, err := o.client.Consumption.GetPagesWithContext(o.ctx, &opts)

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

func (o *OctopusClient) getMPNAndSerialForFuelType(fuelType octopusenergy.FuelType) (string, string) {
	switch fuelType {
	case octopusenergy.FuelTypeElectricity:
		return o.conf.ElectricityMPN, o.conf.ElectricitySerial
	case octopusenergy.FuelTypeGas:
		return o.conf.GasMPN, o.conf.GasSerial
	}
	return "", ""
}

type ConsumptionGrouped struct {
	Consumption float64
	FuelType    octopusenergy.FuelType
	PeriodStart time.Time
}

func (o *OctopusClient) GetAllConsumption(fueltype octopusenergy.FuelType, groupBy string) (float64, []ConsumptionGrouped, error) {
	epoch := time.Unix(0, 0)

	orderByPeriod := "period"
	mpn, serial := o.getMPNAndSerialForFuelType(fueltype)
	opts := octopusenergy.ConsumptionGetOptions{
		FuelType:     fueltype,
		GroupBy:      &groupBy,
		MPN:          mpn,
		OrderBy:      &orderByPeriod,
		PeriodFrom:   &epoch,
		SerialNumber: serial,
	}
	c := make(chan *ConsumptionResponse)

	errGroup := new(errgroup.Group)

	errGroup.Go(func() error {
		err := o.getConsumption(opts, c)
		return err
	})

	var total float64
	var outputGrouped []ConsumptionGrouped

	for val := range c {
		log.WithFields(log.Fields{
			"consumption":      val.Consumption,
			"cumulative_total": total,
			"fuel_type":        fueltype,
		}).Debugf("Got consumption")
		outputGrouped = append(outputGrouped, ConsumptionGrouped{val.Consumption, val.FuelType, val.Period})
		total += val.Consumption
	}

	return total, outputGrouped, errGroup.Wait()
}

func (o *OctopusClient) GetConsumption(wg *sync.WaitGroup, grp *errgroup.Group, electricityPeriod *time.Time, gasPeriod *time.Time, c chan<- *ConsumptionResponse) {
	var orderByPeriod = "period"

	log.Debugf("Getting consumption since %s (electricity), %s (gas)", electricityPeriod, gasPeriod)
	if o.conf.ElectricityMPN != "" {
		wg.Add(1)
	}
	if o.conf.GasMPN != "" {
		wg.Add(1)
	}

	if o.conf.ElectricityMPN != "" {
		grp.Go(func() error {
			defer wg.Done()
			opts := octopusenergy.ConsumptionGetOptions{
				MPN:          o.conf.ElectricityMPN,
				SerialNumber: o.conf.ElectricitySerial,
				FuelType:     octopusenergy.FuelTypeElectricity,
				PeriodFrom:   electricityPeriod,
				OrderBy:      &orderByPeriod,
			}
			return o.getConsumption(opts, c)
		})
	}

	if o.conf.GasMPN != "" {
		grp.Go(func() error {
			defer wg.Done()
			opts := octopusenergy.ConsumptionGetOptions{
				MPN:          o.conf.GasMPN,
				SerialNumber: o.conf.GasSerial,
				FuelType:     octopusenergy.FuelTypeGas,
				PeriodFrom:   gasPeriod,
				OrderBy:      &orderByPeriod,
			}
			return o.getConsumption(opts, c)
		})
	}
}

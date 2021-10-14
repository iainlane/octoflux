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

package influx

import (
	"context"
	"fmt"
	"time"

	influx "github.com/influxdata/influxdb-client-go/v2"
	influxapi "github.com/influxdata/influxdb-client-go/v2/api"
	influxdomain "github.com/influxdata/influxdb-client-go/v2/domain"

	"github.com/iainlane/octoflux/internal/conf"

	log "github.com/sirupsen/logrus"
)

type Influx struct {
	client      influx.Client
	queryClient influxapi.QueryAPI
	writeClient influxapi.WriteAPIBlocking
}

func MakeInfluxClient(ctx context.Context, config *conf.Conf) (Influx, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	client := influx.NewClient(config.InfluxHost, config.InfluxToken)

	ready, err := client.Ready(ctx)
	if err != nil {
		return Influx{}, err
	}

	if !ready {
		return Influx{}, err
	}

	healthcheck, err := client.Health(ctx)
	if err != nil {
		return Influx{}, err
	}

	if healthcheck.Status != influxdomain.HealthCheckStatusPass {
		err := fmt.Errorf("InfluxDB server not healthy")
		return Influx{}, err
	}

	queryClient := client.QueryAPI(config.InfluxOrg)

	var writeClient influxapi.WriteAPIBlocking
	if !config.DryRun {
		writeClient = client.WriteAPIBlocking(config.InfluxOrg, config.InfluxBucket)
	}

	return Influx{client, queryClient, writeClient}, nil
}

func (influx Influx) Close() {
	influx.client.Close()
}

func (ifx Influx) GetLastSubmission(ctx context.Context, config *conf.Conf, fuelType string) (time.Time, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*10)
	defer cancel()
	result, err := ifx.queryClient.Query(ctx,
		fmt.Sprintf(`from(bucket:"octopus")
		|> range(start: -1y)
		|> filter(fn: (r) => r._measurement == "consumption" and
		                     r.fuel_type == "%s")
		|> group(columns: ["_measurement"])
		|> sort(columns: ["_time"])
		|> last()`, fuelType))

	if err != nil {
		return time.Time{}, err
	}

	if !result.Next() {
		log.Info("No last submission found")
		return time.Time{}, nil
	}

	if result.Err() != nil {
		log.WithField("fuel_type", fuelType).WithError(err).Error("Error getting last submission")
		return time.Time{}, err
	}

	record := result.Record()
	lastTime := record.Time().Add(1 * time.Second)
	log.WithFields(log.Fields{
		"fuel_type":       fuelType,
		"last_submission": lastTime,
	}).Debug("Got last submission")

	return lastTime, nil
}

func (ifx Influx) SubmitConsumption(ctx context.Context, config *conf.Conf, energyType string, value float64, t time.Time) error {
	if config.DryRun {
		log.WithFields(log.Fields{
			"energy_type": energyType,
			"value":       value,
			"time":        t,
		}).Info("[dry-run] Would submit consumption")
		return nil
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*1)
	defer cancel()

	point := influx.NewPoint(
		"consumption",
		map[string]string{
			"fuel_type": energyType,
		},
		map[string]interface{}{
			"consumption": value,
		},
		t,
	)
	err := ifx.writeClient.WritePoint(ctx, point)

	return err
}

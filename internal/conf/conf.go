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

package conf

type Conf struct {
	Debug  bool `short:"d" long:"debug" description:"Turn on debug output"`
	DryRun bool `short:"n" long:"dry-run" description:"Dry run: don't actually write anything, but print what would be written"`

	InfluxHost   string `short:"i" long:"influx-host" description:"InfluxDB host" required:"true" env:"INFLUX_HOST"`
	InfluxOrg    string `short:"o" long:"influx-org" description:"InfluxDB organization" required:"true" env:"INFLUX_ORG"`
	InfluxBucket string `short:"b" long:"influx-bucket" description:"InfluxDB bucket" required:"true" env:"INFLUX_BUCKET"`
	InfluxToken  string `short:"t" long:"influx-token" description:"InfluxDB token" required:"true" env:"INFLUX_TOKEN"`

	OctopusAPIKey string `short:"k" long:"octopus-api-key" description:"Octopus API key" required:"true" env:"OCTOPUS_API_KEY"`

	ElectricityMPN    string `long:"electricity-mpn" description:"MPN for electricity" required:"false" env:"ELECTRICITY_MPN"`
	ElectricitySerial string `long:"electricity-serial" description:"Serial for electricity" required:"false" env:"ELECTRICITY_SERIAL"`

	GasMPN    string `long:"gas-mpn" description:"MPN for gas" required:"false" env:"GAS_MPN"`
	GasSerial string `long:"gas-serial" description:"Serial for gas" required:"false" env:"GAS_SERIAL"`
}

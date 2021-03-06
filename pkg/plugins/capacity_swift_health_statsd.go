/*******************************************************************************
*
* Copyright 2017 SAP SE
*
* Licensed under the Apache License, Version 2.0 (the "License");
* you may not use this file except in compliance with the License.
* You should have received a copy of the License along with this
* program. If not, you may obtain a copy of the License at
*
*     http://www.apache.org/licenses/LICENSE-2.0
*
* Unless required by applicable law or agreed to in writing, software
* distributed under the License is distributed on an "AS IS" BASIS,
* WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
* See the License for the specific language governing permissions and
* limitations under the License.
*
*******************************************************************************/

package plugins

import (
	"time"

	"github.com/gophercloud/gophercloud"
	"github.com/prometheus/client_golang/api/prometheus"
	"github.com/prometheus/common/model"
	"github.com/sapcc/limes/pkg/limes"
	"github.com/sapcc/limes/pkg/util"
	"golang.org/x/net/context"
)

type capacitySwiftHealthStatsdPlugin struct {
	cfg limes.CapacitorConfiguration
}

func init() {
	limes.RegisterCapacityPlugin(func(c limes.CapacitorConfiguration) limes.CapacityPlugin {
		return &capacitySwiftHealthStatsdPlugin{c}
	})
}

//Client relates to the prometheus client
//requires the url to prometheus à la "http<s>://localhost<:9090>"
//in our case even without port
func Client(prometheusAPIURL string) (prometheus.QueryAPI, error) {

	config := prometheus.Config{
		Address:   prometheusAPIURL,
		Transport: prometheus.DefaultTransport,
	}
	client, err := prometheus.New(config)
	if err != nil {
		util.LogDebug("Could not create Prometheus client with URL: %s", prometheusAPIURL)
		return nil, err
	}
	return prometheus.NewQueryAPI(client), nil
}

//ID implements the limes.CapacityPlugin interface.
func (p *capacitySwiftHealthStatsdPlugin) ID() string {
	return "swift-health-statsd"
}

//Scrape implements the limes.CapacityPlugin interface.
func (p *capacitySwiftHealthStatsdPlugin) Scrape(provider *gophercloud.ProviderClient) (map[string]map[string]uint64, error) {

	var prometheusQuery = "min(swift_cluster_storage_capacity_bytes_gauge < inf)"
	var prometheusAPIURL = "https://localhost:9090"
	if p.cfg.Swift.PrometheusAPIURL != "" {
		prometheusAPIURL = p.cfg.Swift.PrometheusAPIURL
	}

	client, err := Client(prometheusAPIURL)
	if err != nil {
		return nil, err
	}

	var value model.Value
	var resultVector model.Vector
	var capacity = map[string]uint64{}
	var adjustmentFactor = 1.0

	value, err = client.Query(context.Background(), prometheusQuery, time.Now())
	if err != nil {
		util.LogError("Could not get value for query %s from Prometheus %s.", prometheusQuery, prometheusAPIURL)
		return nil, err
	}
	resultVector, ok := value.(model.Vector)
	if !ok {
		util.LogError("Could not get value for query %s from Prometheus due to type mismatch.", prometheusQuery)
		return nil, nil
	}

	if p.cfg.Swift.AdjustmentFactor != 0 {
		adjustmentFactor = p.cfg.Swift.AdjustmentFactor
	}

	if resultVector.Len() != 0 {
		capacity["capacity"] = uint64(float64(resultVector[0].Value) * adjustmentFactor)
	}

	//returns something like
	//"object-store": {
	//	"capacity": capacity,
	//}
	return map[string]map[string]uint64{
		"object-store": capacity,
	}, nil

}

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
	"github.com/gophercloud/gophercloud"
	"github.com/gophercloud/gophercloud/openstack"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/limits"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/extensions/quotasets"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/flavors"
	"github.com/gophercloud/gophercloud/openstack/compute/v2/servers"
	"github.com/gophercloud/gophercloud/pagination"
	"github.com/sapcc/limes/pkg/limes"
)

type novaPlugin struct {
	cfg             limes.ServiceConfiguration
	scrapeInstances bool
	flavors         map[string]*flavors.Flavor
}

var novaResources = []limes.ResourceInfo{
	{
		Name: "cores",
		Unit: limes.UnitNone,
	},
	{
		Name: "instances",
		Unit: limes.UnitNone,
	},
	{
		Name: "ram",
		Unit: limes.UnitMebibytes,
	},
}

func init() {
	limes.RegisterQuotaPlugin(func(c limes.ServiceConfiguration, scrapeSubresources map[string]bool) limes.QuotaPlugin {
		return &novaPlugin{
			cfg:             c,
			scrapeInstances: scrapeSubresources["instances"],
		}
	})
}

//ServiceInfo implements the limes.QuotaPlugin interface.
func (p *novaPlugin) ServiceInfo() limes.ServiceInfo {
	return limes.ServiceInfo{
		Type:        "compute",
		ProductName: "nova",
		Area:        "compute",
	}
}

//Resources implements the limes.QuotaPlugin interface.
func (p *novaPlugin) Resources() []limes.ResourceInfo {
	return novaResources
}

func (p *novaPlugin) Client(provider *gophercloud.ProviderClient) (*gophercloud.ServiceClient, error) {
	return openstack.NewComputeV2(provider,
		gophercloud.EndpointOpts{Availability: gophercloud.AvailabilityPublic},
	)
}

//Scrape implements the limes.QuotaPlugin interface.
func (p *novaPlugin) Scrape(provider *gophercloud.ProviderClient, domainUUID, projectUUID string) (map[string]limes.ResourceData, error) {
	client, err := p.Client(provider)
	if err != nil {
		return nil, err
	}

	quotas, err := quotasets.Get(client, projectUUID).Extract()
	if err != nil {
		return nil, err
	}

	limits, err := limits.Get(client, limits.GetOpts{TenantID: projectUUID}).Extract()
	if err != nil {
		return nil, err
	}

	var instanceData []interface{}
	if p.scrapeInstances {
		listOpts := novaServerListOpts{
			AllTenants: true,
			TenantID:   projectUUID,
		}

		err := servers.List(client, listOpts).EachPage(func(page pagination.Page) (bool, error) {
			instances, err := servers.ExtractServers(page)
			if err != nil {
				return false, err
			}

			for _, instance := range instances {
				subResource := map[string]interface{}{
					"id":     instance.ID,
					"name":   instance.Name,
					"status": instance.Status,
				}
				flavor, err := p.getFlavor(client, instance.Flavor["id"].(string))
				if err == nil {
					subResource["vcpu"] = flavor.VCPUs
					subResource["ram"] = limes.ValueWithUnit{
						Value: uint64(flavor.RAM),
						Unit:  limes.UnitMebibytes,
					}
					subResource["disk"] = limes.ValueWithUnit{
						Value: uint64(flavor.Disk),
						Unit:  limes.UnitGibibytes,
					}
				}
				instanceData = append(instanceData, subResource)
			}
			return true, nil
		})
		if err != nil {
			return nil, err
		}
	}

	return map[string]limes.ResourceData{
		"cores": {
			Quota: int64(quotas.Cores),
			Usage: uint64(limits.Absolute.TotalCoresUsed),
		},
		"instances": {
			Quota:        int64(quotas.Instances),
			Usage:        uint64(limits.Absolute.TotalInstancesUsed),
			Subresources: instanceData,
		},
		"ram": {
			Quota: int64(quotas.Ram),
			Usage: uint64(limits.Absolute.TotalRAMUsed),
		},
	}, nil
}

//SetQuota implements the limes.QuotaPlugin interface.
func (p *novaPlugin) SetQuota(provider *gophercloud.ProviderClient, domainUUID, projectUUID string, quotas map[string]uint64) error {
	client, err := p.Client(provider)
	if err != nil {
		return err
	}

	return quotasets.Update(client, projectUUID, quotasets.UpdateOpts{
		Cores:     makeIntPointer(int(quotas["cores"])),
		Instances: makeIntPointer(int(quotas["instances"])),
		Ram:       makeIntPointer(int(quotas["ram"])),
	}).Err
}

//Getting and caching flavor details
//Changing a flavor is not supported from OpenStack, so no invalidating of the cache needed
//Acces to the map is not thread safe
func (p *novaPlugin) getFlavor(client *gophercloud.ServiceClient, flavorID string) (*flavors.Flavor, error) {
	if p.flavors == nil {
		p.flavors = make(map[string]*flavors.Flavor)
	}

	if flavor, ok := p.flavors[flavorID]; ok {
		return flavor, nil
	}

	flavor, err := flavors.Get(client, flavorID).Extract()
	if err == nil {
		p.flavors[flavorID] = flavor
	}
	return flavor, err
}

func makeIntPointer(value int) *int {
	return &value
}

type novaServerListOpts struct {
	AllTenants bool   `q:"all_tenants"`
	TenantID   string `q:"tenant_id"`
}

func (opts novaServerListOpts) ToServerListQuery() (string, error) {
	q, err := gophercloud.BuildQueryString(opts)
	return q.String(), err
}

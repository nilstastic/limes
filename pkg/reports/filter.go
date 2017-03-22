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

package reports

import "net/http"

//Filter describes query parameters that can be sent to various GET endpoints
//to filter the reports generated by this package.
type Filter struct {
	isServiceType  map[string]bool
	isResourceName map[string]bool
}

//ReadFilter extracts a Filter from the given Request.
func ReadFilter(r *http.Request) Filter {
	var f Filter
	queryValues := r.URL.Query()
	if services, ok := queryValues["service"]; ok {
		f.isServiceType = make(map[string]bool)
		for _, srv := range services {
			f.isServiceType[srv] = true
		}
	}
	if resources, ok := queryValues["resource"]; ok {
		f.isResourceName = make(map[string]bool)
		for _, res := range resources {
			f.isResourceName[res] = true
		}
	}
	return f
}

//MatchesService applies the filter to the given service.
func (f Filter) MatchesService(serviceType string) bool {
	return f.isServiceType == nil || f.isServiceType[serviceType]
}

//MatchesResource applies the filter to the given resource.
func (f Filter) MatchesResource(resourceName string) bool {
	return f.isResourceName == nil || f.isResourceName[resourceName]
}
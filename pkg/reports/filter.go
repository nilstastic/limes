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
	ServiceTypes  []string
	ResourceNames []string
}

//ReadFilter extracts a Filter from the given Request.
func ReadFilter(r *http.Request) Filter {
	queryValues := r.URL.Query()
	return Filter{
		ServiceTypes:  queryValues["service"],
		ResourceNames: queryValues["resource"],
	}
}

//ApplyTo appends the Filter to a `fields` map that will later be passed to
//db.BuildSimpleWhereClause.
func (f Filter) ApplyTo(fields map[string]interface{}, serviceTableName, resourceTableName string) {
	if len(f.ServiceTypes) > 0 {
		fields[serviceTableName+".type"] = f.ServiceTypes
	}
	if len(f.ResourceNames) > 0 {
		fields[resourceTableName+".type"] = f.ResourceNames
	}
}

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

package collector

import (
	"testing"
	"time"

	"github.com/sapcc/limes/pkg/db"
	"github.com/sapcc/limes/pkg/test"
)

func Test_Consistency(t *testing.T) {
	test.ResetTime()
	cluster := keystoneTestCluster(t)
	c := Collector{
		Cluster:  cluster,
		Plugin:   nil,
		LogError: t.Errorf,
		TimeNow:  test.TimeNow,
		Once:     true,
	}

	//run ScanDomains once to establish a baseline
	_, err := ScanDomains(cluster, ScanDomainsOpts{})
	if err != nil {
		t.Errorf("ScanDomains failed: %v", err)
	}
	test.AssertDBContent(t, "fixtures/scandomains1.sql")

	//check that CheckConsistency() is satisfied with the
	//{domain,project}_services created by ScanDomains(), but adds
	//cluster_services entries
	c.CheckConsistency()
	test.AssertDBContent(t, "fixtures/checkconsistency0.sql")

	//remove some *_services entries
	_, err = db.DB.Exec(`DELETE FROM cluster_services WHERE type = ?`, "shared")
	if err != nil {
		t.Error(err)
	}
	_, err = db.DB.Exec(`DELETE FROM domain_services WHERE type = ?`, "unshared")
	if err != nil {
		t.Error(err)
	}
	_, err = db.DB.Exec(`DELETE FROM project_services WHERE type = ?`, "shared")
	if err != nil {
		t.Error(err)
	}
	//add some useless *_services entries
	epoch := time.Unix(0, 0)
	err = db.DB.Insert(&db.ClusterService{
		ClusterID: "shared",
		Type:      "whatever",
		ScrapedAt: &epoch,
	})
	if err != nil {
		t.Error(err)
	}
	err = db.DB.Insert(&db.ClusterService{
		//this one is particularly interesting: the "shared" service type exists,
		//but it may not be unshared as it is here (i.e. ClusterID should be "shared")
		ClusterID: "west",
		Type:      "shared",
		ScrapedAt: &epoch,
	})
	if err != nil {
		t.Error(err)
	}
	err = db.DB.Insert(&db.DomainService{
		DomainID: 1,
		Type:     "whatever",
	})
	if err != nil {
		t.Error(err)
	}
	err = db.DB.Insert(&db.ProjectService{
		ProjectID: 1,
		Type:      "whatever",
	})
	if err != nil {
		t.Error(err)
	}
	test.AssertDBContent(t, "fixtures/checkconsistency1.sql")

	//check that CheckConsistency() brings everything back into a nice state (BUT
	//cluster_service shared/whatever is left as it is even though this service
	//is not enabled, because CheckConsistency() only looks at one cluster and
	//cannot know whether another cluster has this service enabled)
	c.CheckConsistency()
	test.AssertDBContent(t, "fixtures/checkconsistency2.sql")
}

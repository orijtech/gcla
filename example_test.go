// Copyright 2017 orijtech. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package gcla_test

import (
	"log"

	"github.com/orijtech/gcla/v3"
)

func Example_client_SubscribeToEvents() {
	client, err := gcla.NewClientFromEnv()
	if err != nil {
		log.Fatal(err)
	}

	subscr, err := client.SubscribeToRepo(&gcla.RepoSubscribeRequest{
		Repo:  "gcla",
		Owner: "orijtech",
		HookSubscription: &gcla.SubscribeRequest{
			Name:   "test",
			Active: true,
			Events: []gcla.Event{
				gcla.EventIssues,
				gcla.EventPush,
				gcla.EventPullRequest,
			},
			Config: &gcla.PayloadConfig{
				URL:         "https://hooks.orijtech/gcla-test",
				ContentType: gcla.JSON,
			},
		},
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("subscription: %#v\n", subscr)
}

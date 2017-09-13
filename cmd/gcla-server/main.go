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

package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"

	"github.com/orijtech/gcla/v3"
)

func main() {
	var port int
	flag.IntVar(&port, "port", 9889, "the port on which the server runs")
	flag.Parse()

	addr := fmt.Sprintf(":%d", port)
	http.HandleFunc("/", handleWebhooks)
	http.HandleFunc("/ping", pong)

	if err := http.ListenAndServe(addr, nil); err != nil {
		log.Fatal(err)
	}
}

func handleWebhooks(w http.ResponseWriter, r *http.Request) {
}

func parseRequest(req *http.Request, savPtr interface{}) error {
	defer req.Body.Close()
	blob, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return err
	}
	return json.Unmarshal(blob, savPtr)
}

func pong(w http.ResponseWriter, r *http.Request) {
	pPayload := new(pingPayload)
	if err := parseRequest(r, pPayload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
}

type pingPayload struct {
	Zen    string     `json:"zen,omitempty"`
	HookID string     `json:"hook_id,omitempty"`
	Hook   *gcla.Hook `json:"hook,omitempty"`
}

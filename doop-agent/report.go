/*******************************************************************************
*
* Copyright 2021 SAP SE
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

package main

import (
	"context"
	"encoding/json"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var gvTemplatesV1Beta1 = schema.GroupVersion{
	Group:   "templates.gatekeeper.sh",
	Version: "v1beta1",
}

//ClientSet provides access to the Gatekeeper API groups in k8s.
type ClientSet struct {
	templatesV1Beta1 dynamic.Interface
}

//NewClientSet builds a ClientSet.
func NewClientSet(cfg *rest.Config) ClientSet {
	return ClientSet{
		templatesV1Beta1: newClientTemplatesV1Beta1(cfg),
	}
}

func newClientTemplatesV1Beta1(cfg *rest.Config) dynamic.Interface {
	cfgCloned := dynamic.ConfigFor(cfg)
	cfgCloned.GroupVersion = &gvTemplatesV1Beta1
	cfgCloned.APIPath = "/apis"
	client, err := dynamic.NewForConfig(cfgCloned)
	must("build templates.gatekeeper.sh/v1beta1 client", err)
	return client
}

type ConstraintTemplate struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	//TODO: Spec, Status
}

func (cs ClientSet) ListConstraintTemplates(ctx context.Context) []ConstraintTemplate {
	gvr := gvTemplatesV1Beta1.WithResource("constrainttemplates")
	list, err := cs.templatesV1Beta1.Resource(gvr).List(ctx, metav1.ListOptions{})
	must("list ConstraintTemplates", err)
	result := make([]ConstraintTemplate, len(list.Items))
	for idx, item := range list.Items {
		//convert item from unstructured.Unstructured to ConstraintTemplate through a JSON roundtrip
		jsonBytes, err := json.Marshal(item.Object)
		must("encode ConstraintTemplate as JSON", err)
		err = json.Unmarshal(jsonBytes, &result[idx])
		must("decode ConstraintTemplate from JSON", err)
	}
	return result
}

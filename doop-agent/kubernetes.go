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
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
)

var (
	gvConstraintsV1Beta1 = schema.GroupVersion{
		Group:   "constraints.gatekeeper.sh",
		Version: "v1beta1",
	}
	gvTemplatesV1Beta1 = schema.GroupVersion{
		Group:   "templates.gatekeeper.sh",
		Version: "v1beta1",
	}
)

// ClientSet provides access to the Gatekeeper API groups in k8s.
type ClientSet struct {
	constraintsV1Beta1 dynamic.Interface
	templatesV1Beta1   dynamic.Interface
}

// NewClientSet builds a ClientSet.
func NewClientSet(cfg *rest.Config) ClientSet {
	newClient := func(gv schema.GroupVersion) dynamic.Interface {
		cfgCloned := dynamic.ConfigFor(cfg)
		cfgCloned.GroupVersion = &gv
		cfgCloned.APIPath = "/apis"
		client, err := dynamic.NewForConfig(cfgCloned)
		must(fmt.Sprintf("build %s/%s client", gv.Group, gv.Version), err)
		return client
	}

	return ClientSet{
		constraintsV1Beta1: newClient(gvConstraintsV1Beta1),
		templatesV1Beta1:   newClient(gvTemplatesV1Beta1),
	}
}

// ConstraintTemplate is the unpacked form of `kind: ConstraintTemplate`.
type ConstraintTemplate struct {
	Metadata metav1.ObjectMeta `json:"metadata"`
	Spec     struct {
		CRD struct {
			Spec struct {
				Names struct {
					Kind string `json:"kind"`
				} `json:"names"`
			} `json:"spec"`
		} `json:"crd"`
	} `json:"spec"`
	Status struct {
		//TODO We could report Rego parse errors.
		Created bool `json:"created"`
	} `json:"status"`
}

// ListConstraintTemplates lists all constraint templates.
func (cs ClientSet) ListConstraintTemplates(ctx context.Context) []ConstraintTemplate {
	gvr := gvTemplatesV1Beta1.WithResource("constrainttemplates")
	list, err := cs.templatesV1Beta1.Resource(gvr).List(ctx, metav1.ListOptions{})
	must("list ConstraintTemplates", err)

	//convert items from unstructured.Unstructured to ConstraintTemplate through a JSON roundtrip
	result := make([]ConstraintTemplate, len(list.Items))
	for idx, item := range list.Items {
		jsonBytes, err := json.Marshal(item.Object)
		must("encode ConstraintTemplate as JSON", err)
		err = json.Unmarshal(jsonBytes, &result[idx])
		must("decode ConstraintTemplate from JSON", err)
	}
	return result
}

// ConstraintConfig is the unpacked form of any object in the `constraints.gatekeeper.sh` API group.
type ConstraintConfig struct {
	Kind     string            `json:"kind"`
	Metadata metav1.ObjectMeta `json:"metadata"`
	Status   struct {
		//TODO: We could parse `json:"byPod"` to report on whether Gatekeeper is functioning correctly.
		AuditTimestamp string                `json:"auditTimestamp"`
		Violations     []ConstraintViolation `json:"violations"`
	} `json:"status"`
}

// ConstraintViolation appears in type ConstraintConfig.
type ConstraintViolation struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Message           string `json:"message"`
	EnforcementAction string `json:"enforcementAction"`
}

// ListConstraintConfigs lists all constraint configs for a given template.
func (cs ClientSet) ListConstraintConfigs(ctx context.Context, tmpl ConstraintTemplate) []ConstraintConfig {
	//The following will not work unless the respective CRD was created.
	if !tmpl.Status.Created {
		return nil
	}

	gvr := gvConstraintsV1Beta1.WithResource(tmpl.Metadata.Name)
	list, err := cs.constraintsV1Beta1.Resource(gvr).List(ctx, metav1.ListOptions{})
	must("list ConstraintConfigs for "+tmpl.Spec.CRD.Spec.Names.Kind, err)

	//convert items from unstructured.Unstructured to ConstraintConfig through a JSON roundtrip
	result := make([]ConstraintConfig, len(list.Items))
	for idx, item := range list.Items {
		jsonBytes, err := json.Marshal(item.Object)
		must("encode ConstraintConfig as JSON", err)
		err = json.Unmarshal(jsonBytes, &result[idx])
		must("decode ConstraintConfig from JSON", err)
	}
	return result
}

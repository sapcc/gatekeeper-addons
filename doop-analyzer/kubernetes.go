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

	k8sinternal "github.com/sapcc/gatekeeper-addons/internal/kubernetes"
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

// ClientSetInterface contains the methods that ClientSet provides. This
// interface can be mocked in unit tests.
type ClientSetInterface interface {
	ListConstraintTemplates(ctx context.Context) ([]ConstraintTemplate, error)
	ListConstraints(ctx context.Context, tmpl ConstraintTemplate) ([]Constraint, error)
}

// NewClientSet builds a ClientSet.
func NewClientSet(cfg Configuration) (cs ClientSet, err error) {
	var kcfg *rest.Config
	if cfg.Kubernetes.KubeconfigPath == "" {
		kcfg, err = rest.InClusterConfig()
	} else {
		kcfg, err = k8sinternal.NewClientConfig(cfg.Kubernetes.KubeconfigPath, cfg.Kubernetes.Context).ClientConfig()
	}
	if err != nil {
		return ClientSet{}, fmt.Errorf("cannot assemble Kubernetes client config: %w", err)
	}

	newClient := func(gv schema.GroupVersion) (dynamic.Interface, error) {
		dcfg := dynamic.ConfigFor(kcfg)
		dcfg.GroupVersion = &gv
		dcfg.APIPath = "/apis"
		client, err := dynamic.NewForConfig(dcfg)
		if err != nil {
			err = fmt.Errorf("build %s/%s client: %w", gv.Group, gv.Version, err)
		}
		return client, err
	}

	cs.constraintsV1Beta1, err = newClient(gvConstraintsV1Beta1)
	if err != nil {
		return ClientSet{}, err
	}
	cs.templatesV1Beta1, err = newClient(gvTemplatesV1Beta1)
	return cs, err
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
		// TODO We could report Rego parse errors.
		Created bool `json:"created"`
	} `json:"status"`
}

// ListConstraintTemplates lists all constraint templates.
func (cs ClientSet) ListConstraintTemplates(ctx context.Context) ([]ConstraintTemplate, error) {
	gvr := gvTemplatesV1Beta1.WithResource("constrainttemplates")
	list, err := cs.templatesV1Beta1.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot list ConstraintTemplates: %w", err)
	}

	// convert items from unstructured.Unstructured to ConstraintTemplate through a JSON roundtrip
	result := make([]ConstraintTemplate, len(list.Items))
	for idx, item := range list.Items {
		jsonBytes, err := json.Marshal(item.Object)
		if err != nil {
			return nil, fmt.Errorf("cannot encode ConstraintTemplate as JSON: %w", err)
		}
		err = json.Unmarshal(jsonBytes, &result[idx])
		if err != nil {
			return nil, fmt.Errorf("cannot decode ConstraintTemplate from JSON: %w", err)
		}
	}
	return result, nil
}

// Constraint is the unpacked form of any object in the `constraints.gatekeeper.sh` API group.
type Constraint struct {
	Kind     string            `json:"kind"`
	Metadata metav1.ObjectMeta `json:"metadata"`
	Status   struct {
		//TODO: We could parse `json:"byPod"` to report on whether Gatekeeper is functioning correctly.
		AuditTimestamp string                `json:"auditTimestamp"`
		Violations     []ConstraintViolation `json:"violations"`
	} `json:"status"`
}

// ConstraintViolation appears in type Constraint.
type ConstraintViolation struct {
	Kind              string `json:"kind"`
	Name              string `json:"name"`
	Namespace         string `json:"namespace"`
	Message           string `json:"message"`
	EnforcementAction string `json:"enforcementAction"`
}

// ListConstraints lists all constraints for a given template.
func (cs ClientSet) ListConstraints(ctx context.Context, tmpl ConstraintTemplate) ([]Constraint, error) {
	// The following will not work unless the respective CRD was created.
	if !tmpl.Status.Created {
		return nil, nil
	}

	gvr := gvConstraintsV1Beta1.WithResource(tmpl.Metadata.Name)
	list, err := cs.constraintsV1Beta1.Resource(gvr).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("cannot list constraints for %s: %w", tmpl.Spec.CRD.Spec.Names.Kind, err)
	}

	// convert items from unstructured.Unstructured to Constraint through a JSON roundtrip
	result := make([]Constraint, len(list.Items))
	for idx, item := range list.Items {
		jsonBytes, err := json.Marshal(item.Object)
		if err != nil {
			return nil, fmt.Errorf("cannot encode Constraint as JSON: %w", err)
		}
		err = json.Unmarshal(jsonBytes, &result[idx])
		if err != nil {
			return nil, fmt.Errorf("cannot decode Constraint from JSON: %w", err)
		}
	}
	return result, nil
}

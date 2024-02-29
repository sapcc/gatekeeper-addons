/*******************************************************************************
*
* Copyright 2023 SAP SE
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

package doop

import (
	"encoding/json"
	"strings"

	"github.com/sapcc/gatekeeper-addons/internal/util"
)

// ViolationGroup describes a set of one or more policy violations that follow a common pattern.
type ViolationGroup struct {
	Pattern   Violation   `json:"pattern"`
	Instances []Violation `json:"instances"`
}

// Violation describes a single policy violation, or the common pattern within a ViolationGroup.
type Violation struct {
	Kind           string
	Name           string
	Namespace      string
	Message        string
	ObjectIdentity *util.CowMap[string, string]
	// This field is only set when this Violation appears as a ViolationGroup instance inside an AggregatedReport.
	ClusterName string
}

type serializedViolation struct {
	// All fields are omitempty because we compress ViolationGroups by omitting all fields
	// from instances that are identical to the respective fields in the pattern.
	Kind           string            `json:"kind,omitempty"`
	Name           string            `json:"name,omitempty"`
	Namespace      string            `json:"namespace,omitempty"`
	Message        string            `json:"message,omitempty"`
	ObjectIdentity map[string]string `json:"object_identity,omitempty"`
	// This field is only set when this Violation appears as a ViolationGroup instance inside an AggregatedReport.
	ClusterName string `json:"cluster,omitempty"`
}

// MarshalJSON implements the json.Marshaler interface.
func (v Violation) MarshalJSON() ([]byte, error) {
	return json.Marshal(serializedViolation{
		Kind:           v.Kind,
		Name:           v.Name,
		Namespace:      v.Namespace,
		Message:        v.Message,
		ObjectIdentity: v.ObjectIdentity.GetAll(),
		ClusterName:    v.ClusterName,
	})
}

// UnmarshalJSON implements the json.Unmarshaler interface.
func (v *Violation) UnmarshalJSON(buf []byte) error {
	var s serializedViolation
	err := json.Unmarshal(buf, &s)
	if err != nil {
		return err
	}

	*v = Violation{
		Kind:           s.Kind,
		Name:           s.Name,
		Namespace:      s.Namespace,
		Message:        s.Message,
		ObjectIdentity: util.NewCowMap(s.ObjectIdentity),
		ClusterName:    s.ClusterName,
	}
	return nil
}

// Cloned returns a deep copy of this Violation.
func (v Violation) Cloned() Violation {
	result := v
	result.ObjectIdentity = v.ObjectIdentity.Clone()
	return result
}

// IsEqualTo works like reflect.DeepEqual(), but is faster and thus a better
// fit for hot loops.
func (v Violation) IsEqualTo(other Violation) bool {
	return v.Kind == other.Kind &&
		v.Name == other.Name &&
		v.Namespace == other.Namespace &&
		v.Message == other.Message &&
		v.ObjectIdentity.IsEqual(other.ObjectIdentity) &&
		v.ClusterName == other.ClusterName
}

// DifferenceTo returns a copy of this violation, with all fields cleared out
// that are identical to the pattern.
func (v Violation) DifferenceTo(pattern Violation) Violation {
	result := v
	if result.Kind == pattern.Kind {
		result.Kind = ""
	}
	if result.Name == pattern.Name {
		result.Name = ""
	}
	if result.Namespace == pattern.Namespace {
		result.Namespace = ""
	}
	if result.Message == pattern.Message {
		result.Message = ""
	}
	if result.ObjectIdentity.IsEqual(pattern.ObjectIdentity) {
		result.ObjectIdentity = util.NewCowMap[string, string](nil)
	}
	if result.ClusterName == pattern.ClusterName {
		result.ClusterName = ""
	}
	return result
}

// CompareTo is a three-way compare between violations. As per the usual convention,
// 0 means `v == other`, negative means `v < other`, and positive means `v > other`.
func (v Violation) CompareTo(other Violation) int {
	cmp := strings.Compare(v.Namespace, other.Namespace)
	if cmp != 0 {
		return cmp
	}
	cmp = strings.Compare(v.Name, other.Name)
	if cmp != 0 {
		return cmp
	}
	cmp = strings.Compare(v.Kind, other.Kind)
	if cmp != 0 {
		return cmp
	}
	cmp = strings.Compare(v.Message, other.Message)
	if cmp != 0 {
		return cmp
	}
	return strings.Compare(v.ClusterName, other.ClusterName)
}

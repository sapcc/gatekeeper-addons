/*******************************************************************************
*
* Copyright 2024 SAP SE
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

package util

import (
	"maps"
)

// CowMap is a wrapper for map types that implements copy-on-write semantics.
// This allows safe sharing of maps without having to clone immediately.
type CowMap[K, V comparable] struct {
	value            map[K]V
	isExclusiveOwner bool
}

// NOTE: We cannot implement MarshalJSON/UnmarshalJSON on this type:
// Because the value type is a pointer type (*CowMap), UnmarshalJSON would need
// to be a method on the respective double-pointer type (**CowMap), but Go does
// not allow that.
//
// To work around this issue, MarshalJSON/UnmarshalJSON are implemented on the
// types containing *CowMap members instead (currently only doop.Violation).

// NewCowMap wraps the given map in a copy-on-write container.
func NewCowMap[K, V comparable](val map[K]V) *CowMap[K, V] {
	if val == nil {
		return nil
	}
	return &CowMap[K, V]{value: val, isExclusiveOwner: true}
}

// Get provides read access to the wrapped map, same as `m[key]` on a regular map.
// The value is returned as a shallow copy, so the caller must ensure that it does not
// modify its deeper structures in order to uphold the safety invariant of type CowMap!
func (c *CowMap[K, V]) Get(key K) V {
	if c == nil {
		var zero V
		return zero
	}
	return c.value[key]
}

// GetAll returns a reference to the wrapped map. The caller must ensure that it does
// not modify its contents in order to uphold the safety invariant of type CowMap!
func (c *CowMap[K, V]) GetAll() map[K]V {
	if c == nil {
		return nil
	}
	return c.value
}

// IsEqual checks if both maps have equal contents.
func (c *CowMap[K, V]) IsEqual(other *CowMap[K, V]) bool {
	if c == nil || other == nil {
		return (c == nil) == (other == nil)
	}
	return maps.Equal(c.value, other.value)
}

// Clone returns a cheap copy of this Cow.
func (c *CowMap[K, V]) Clone() *CowMap[K, V] {
	if c == nil {
		return nil
	}
	c.isExclusiveOwner = false
	return &CowMap[K, V]{value: c.value, isExclusiveOwner: false}
}

// Update provides write access to the wrapped value.
// If the value is shared with other copies, a deep copy is made first.
func (c *CowMap[K, V]) Update(action func(map[K]V)) {
	if c == nil {
		panic("CowMap.Update: called on nil map")
	}
	if !c.isExclusiveOwner {
		c.value = maps.Clone(c.value)
		c.isExclusiveOwner = true
	}
	action(c.value)
}

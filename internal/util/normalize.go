// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package util

import (
	"fmt"
	"strconv"
)

// NormalizeRecursively prepares values obtained from yaml.Unmarshal() for being
// processed by json.Marshal().
//
// yaml.Unmarshal() into a generic map will create map[interface{}]interface{}
// members which json.Marshal() cannot process. We need to convert these into
// map[string]interface{} recursively before proceeding.
func NormalizeRecursively(path string, in interface{}) (interface{}, error) {
	switch in := in.(type) {
	case map[interface{}]interface{}:
		out := make(map[string]interface{}, len(in))
		for k, v := range in {
			kn, err := normalizeKey(path, k)
			if err != nil {
				return nil, err
			}
			vn, err := NormalizeRecursively(fmt.Sprintf("%s.%s", path, kn), v)
			if err != nil {
				return nil, err
			}
			out[kn] = vn
		}
		return out, nil
	case []interface{}:
		out := make([]interface{}, len(in))
		for idx, v := range in {
			vn, err := NormalizeRecursively(fmt.Sprintf("%s[%d]", path, idx), v)
			if err != nil {
				return nil, err
			}
			out[idx] = vn
		}
		return out, nil
	default:
		return in, nil
	}
}

func normalizeKey(path string, k interface{}) (string, error) {
	switch k := k.(type) {
	case string:
		return k, nil
	case int:
		// has been observed in the wild in `.data` of a v1/Secret
		return strconv.Itoa(k), nil
	default:
		return "", fmt.Errorf("non-string key at %s: %T %v", path, k, k)
	}
}

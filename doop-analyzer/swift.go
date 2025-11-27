// SPDX-FileCopyrightText: 2023 SAP SE or an SAP affiliate company
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/gophercloud/gophercloud/v2/openstack"
	"github.com/majewsky/schwift/v2"
	"github.com/majewsky/schwift/v2/gopherschwift"
	"github.com/sapcc/go-bits/gophercloudext"

	"github.com/sapcc/gatekeeper-addons/internal/doop"
)

// SwiftConfiguration appears in type Configuration. It also holds the methods
// and state for talking to Swift.
type SwiftConfiguration struct {
	ContainerName string `json:"container_name"`
	ObjectName    string `json:"object_name"`
	// filled by Connect()
	Object *schwift.Object `json:"-"`
}

// Connect initializes the Swift client.
func (s *SwiftConfiguration) Connect(ctx context.Context) error {
	// check provided configuration
	if s.ContainerName == "" {
		return errors.New("missing required configuration value: swift.container_name")
	}
	if s.ObjectName == "" {
		return errors.New("missing required configuration value: swift.object_name")
	}

	// connect to OpenStack
	provider, eo, err := gophercloudext.NewProviderClient(ctx, nil)
	if err != nil {
		return err
	}
	client, err := openstack.NewObjectStorageV1(provider, eo)
	if err != nil {
		return fmt.Errorf("cannot initialize Swift client: %w", err)
	}
	account, err := gopherschwift.Wrap(client, nil)
	if err != nil {
		return fmt.Errorf("cannot initialize Swift account: %w", err)
	}
	swiftContainer, err := account.Container(s.ContainerName).EnsureExists(ctx)
	if err != nil {
		return fmt.Errorf("cannot initialize Swift container: %w", err)
	}
	s.Object = swiftContainer.Object(s.ObjectName)
	return nil
}

// SendReport uploads a processed report to Swift.
func (s *SwiftConfiguration) SendReport(ctx context.Context, report doop.Report) error {
	buf, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("cannot encode report as JSON: %w", err)
	}

	err = s.Object.Upload(ctx, bytes.NewReader(buf), nil, nil)
	if err != nil {
		return fmt.Errorf("cannot upload report to Swift: %w", err)
	}
	return nil
}

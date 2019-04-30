// Copyright 2017 The Prometheus Authors
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package uyuni

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/go-kit/kit/log"
	"github.com/kolo/xmlrpc"
	"github.com/pkg/errors"
	// config_util "github.com/prometheus/common/config"
	"github.com/prometheus/common/model"

	"github.com/prometheus/prometheus/discovery/refresh"
	"github.com/prometheus/prometheus/discovery/targetgroup"
)

const (
	uyuniLabel             = model.MetaLabelPrefix + "uyuni_"
	uyuniLabelEntitlements = uyuniLabel + "entitlements"
)

// DefaultSDConfig is the default Triton SD configuration.
var DefaultSDConfig = SDConfig{
	RefreshInterval: model.Duration(1 * time.Minute),
}

// SDConfig is the configuration for Triton based service discovery.
type SDConfig struct {
	Host            string         `yaml:"host"`
	User            string         `yaml:"username"`
	Pass            string         `yaml:"password"`
	RefreshInterval model.Duration `yaml:"refresh_interval,omitempty"`
}

// Uyuni API Response structures
type clientRef struct {
	Id   int    `xmlrpc:"id"`
	Name string `xmlrpc:"name"`
}

type clientDetail struct {
	Id           int      `xmlrpc:"id"`
	Hostname     string   `xmlrpc:"hostname"`
	Entitlements []string `xmlrpc:"addon_entitlements"`
}

type exporterConfig struct {
	Enabled bool `xmlrpc:"enabled"`
}

type formulaData struct {
	NodeExporter     exporterConfig `xmlrpc:"node_exporter"`
	PostgresExporter exporterConfig `xmlrpc:"postgres_exporter"`
}

// UnmarshalYAML implements the yaml.Unmarshaler interface.
func (c *SDConfig) UnmarshalYAML(unmarshal func(interface{}) error) error {
	*c = DefaultSDConfig
	type plain SDConfig
	err := unmarshal((*plain)(c))
	if err != nil {
		return err
	}
	if c.Host == "" {
		return errors.New("Uyuni configuration requires a Host")
	}
	if c.User == "" {
		return errors.New("Uyuni configuration requires a User")
	}
	if c.Pass == "" {
		return errors.New("Uyuni configuration requires a Password")
	}
	return nil
}

// Attempt to login in SUSE Manager Server and get an auth token
func Login(host string, user string, pass string) (string, error) {
	client, _ := xmlrpc.NewClient(host, nil)
	var result string
	err := client.Call("auth.login", []interface{}{user, pass}, &result)
	return result, err
}

// Logout from SUSE Manager API
func Logout(host string, token string) error {
	client, _ := xmlrpc.NewClient(host, nil)
	err := client.Call("auth.logout", token, nil)
	return err
}

// Get client list
func ListSystems(host string, token string) ([]clientRef, error) {
	client, _ := xmlrpc.NewClient(host, nil)
	var result []clientRef
	err := client.Call("system.listSystems", token, &result)
	return result, err
}

// Get client details
func GetSystemDetails(host string, token string, systemId int) (clientDetail, error) {
	client, _ := xmlrpc.NewClient(host, nil)
	var result clientDetail
	err := client.Call("system.getDetails", []interface{}{token, systemId}, &result)
	return result, err
}

// List client FQDNs
func ListSystemFQDNs(host string, token string, systemId int) ([]string, error) {
	client, _ := xmlrpc.NewClient(host, nil)
	var result []string
	err := client.Call("system.listFqdns", []interface{}{token, systemId}, &result)
	return result, err
}

// Get formula data for a given system
func getSystemFormulaData(host string, token string, systemId int, formulaName string) (formulaData, error) {
	client, _ := xmlrpc.NewClient(host, nil)
	var result formulaData
	err := client.Call("formula.getSystemFormulaData", []interface{}{token, systemId, formulaName}, &result)
	return result, err
}

// Discovery periodically performs Uyuni API requests. It implements
// the Discoverer interface.
type Discovery struct {
	*refresh.Discovery
	client   *http.Client
	interval time.Duration
	sdConfig *SDConfig
}

// NewDiscovery returns a new file discovery for the given paths.
func NewDiscovery(conf *SDConfig, logger log.Logger) (*Discovery, error) {
	d := &Discovery{
		interval: time.Duration(conf.RefreshInterval),
		sdConfig: conf,
	}
	d.Discovery = refresh.NewDiscovery(
		logger,
		"uyuni",
		time.Duration(conf.RefreshInterval),
		d.refresh,
	)
	return d, nil
}

func (d *Discovery) refresh(ctx context.Context) ([]*targetgroup.Group, error) {

	config := d.sdConfig

	apiUrl := "http://" + config.Host + "/rpc/api"
	token, err := Login(apiUrl, config.User, config.Pass)
	if err != nil {
		fmt.Printf("ERROR - Unable to login to SUSE Manager API: %v\n", err)
		// return err;
	}

	fmt.Printf("DEBUG: tokens: %v\n", token)

	tg := &targetgroup.Group{
		// Source: endpoint,
	}

	// labels[model.AddressLabel] = model.LabelValue(addr)

	// if len(container.Groups) > 0 {
	// 	name := "," + strings.Join(container.Groups, ",") + ","
	// 	labels[uyuniLabelEntitlements] = model.LabelValue(name)
	// }

	// tg.Targets = append(tg.Targets, labels)

	return []*targetgroup.Group{tg}, nil
}

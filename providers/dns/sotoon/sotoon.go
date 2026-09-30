// Package sotoon implements a DNS provider for solving the DNS-01 challenge using Sotoon DNS.
package sotoon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/go-acme/lego/v5/challenge"
	"github.com/go-acme/lego/v5/challenge/dns01"
	"github.com/go-acme/lego/v5/platform/env"
	"github.com/go-acme/lego/v5/providers/dns/internal/clientdebug"
	"github.com/go-acme/lego/v5/providers/dns/sotoon/internal"
)

// Environment variables names.
const (
	envNamespace = "SOTOON_"

	EnvToken         = envNamespace + "TOKEN"
	EnvWorkspaceUUID = envNamespace + "WORKSPACE_UUID"

	EnvTTL                = envNamespace + "TTL"
	EnvPropagationTimeout = envNamespace + "PROPAGATION_TIMEOUT"
	EnvPollingInterval    = envNamespace + "POLLING_INTERVAL"
	EnvHTTPTimeout        = envNamespace + "HTTP_TIMEOUT"
	EnvSequenceInterval   = envNamespace + "SEQUENCE_INTERVAL"
)

var _ challenge.ProviderTimeout = (*DNSProvider)(nil)

// Config is used to configure the creation of the DNSProvider.
type Config struct {
	Token         string
	WorkspaceUUID string

	PropagationTimeout time.Duration
	PollingInterval    time.Duration
	SequenceInterval   time.Duration
	TTL                int
	HTTPClient         *http.Client
}

// NewDefaultConfig returns a default configuration for the DNSProvider.
func NewDefaultConfig() *Config {
	return &Config{
		TTL:                env.GetOrDefaultInt(EnvTTL, dns01.DefaultTTL),
		PropagationTimeout: env.GetOrDefaultSecond(EnvPropagationTimeout, 120*time.Second),
		PollingInterval:    env.GetOrDefaultSecond(EnvPollingInterval, dns01.DefaultPollingInterval),
		SequenceInterval:   env.GetOrDefaultSecond(EnvSequenceInterval, time.Second),
		HTTPClient: &http.Client{
			Timeout: env.GetOrDefaultSecond(EnvHTTPTimeout, 30*time.Second),
		},
	}
}

// DNSProvider implements the challenge.Provider interface.
type DNSProvider struct {
	config *Config
	client *internal.Client

	mu sync.Mutex
}

// NewDNSProvider returns a DNSProvider instance configured for Sotoon.
// Credentials must be passed in the environment variables:
// SOTOON_TOKEN and SOTOON_WORKSPACE_UUID.
func NewDNSProvider() (*DNSProvider, error) {
	values, err := env.Get(EnvToken, EnvWorkspaceUUID)
	if err != nil {
		return nil, fmt.Errorf("sotoon: %w", err)
	}

	config := NewDefaultConfig()
	config.Token = values[EnvToken]
	config.WorkspaceUUID = values[EnvWorkspaceUUID]

	return NewDNSProviderConfig(config)
}

// NewDNSProviderConfig return a DNSProvider instance configured for Sotoon.
func NewDNSProviderConfig(config *Config) (*DNSProvider, error) {
	if config == nil {
		return nil, errors.New("sotoon: the configuration of the DNS provider is nil")
	}

	client, err := internal.NewClient(config.Token, config.WorkspaceUUID)
	if err != nil {
		return nil, fmt.Errorf("sotoon: %w", err)
	}

	if config.HTTPClient != nil {
		client.HTTPClient = config.HTTPClient
	}

	client.HTTPClient = clientdebug.Wrap(client.HTTPClient)

	return &DNSProvider{
		config: config,
		client: client,
	}, nil
}

// Present creates a TXT record using the specified parameters.
func (d *DNSProvider) Present(ctx context.Context, domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(ctx, domain, keyAuth)

	d.mu.Lock()
	defer d.mu.Unlock()

	zone, err := d.findZone(ctx, info.EffectiveFQDN)
	if err != nil {
		return fmt.Errorf("sotoon: %w", err)
	}

	subDomain, err := dns01.ExtractSubDomain(info.EffectiveFQDN, zone.Spec.Origin)
	if err != nil {
		return fmt.Errorf("sotoon: %w", err)
	}

	current, err := d.client.GetDomainZone(ctx, zone.Metadata.Name)
	if err != nil {
		return fmt.Errorf("sotoon: get domain zone: %w", err)
	}

	records := append([]internal.Record(nil), current.Spec.Records[subDomain]...)

	for _, record := range records {
		if record.TXT == info.Value {
			return nil
		}
	}

	records = append(records, internal.Record{
		TXT:  info.Value,
		Type: "TXT",
		TTL:  d.config.TTL,
	})

	patch := internal.DomainZonePatch{
		Spec: internal.SpecPatch{
			Records: map[string]any{
				subDomain: records,
			},
		},
	}

	_, err = d.client.PatchDomainZone(ctx, zone.Metadata.Name, patch)
	if err != nil {
		return fmt.Errorf("sotoon: add TXT record: fqdn=%s: %w", info.EffectiveFQDN, err)
	}

	return nil
}

// CleanUp removes the TXT record matching the specified parameters.
func (d *DNSProvider) CleanUp(ctx context.Context, domain, _, keyAuth string) error {
	info := dns01.GetChallengeInfo(ctx, domain, keyAuth)

	d.mu.Lock()
	defer d.mu.Unlock()

	zone, err := d.findZone(ctx, info.EffectiveFQDN)
	if err != nil {
		return fmt.Errorf("sotoon: %w", err)
	}

	subDomain, err := dns01.ExtractSubDomain(info.EffectiveFQDN, zone.Spec.Origin)
	if err != nil {
		return fmt.Errorf("sotoon: %w", err)
	}

	current, err := d.client.GetDomainZone(ctx, zone.Metadata.Name)
	if err != nil {
		return fmt.Errorf("sotoon: get domain zone: %w", err)
	}

	existing := current.Spec.Records[subDomain]
	if len(existing) == 0 {
		return nil
	}

	var remaining []internal.Record

	for _, record := range existing {
		if record.TXT != info.Value {
			remaining = append(remaining, record)
		}
	}

	if len(remaining) == len(existing) {
		return nil
	}

	var value any
	if len(remaining) == 0 {
		value = nil
	} else {
		value = remaining
	}

	patch := internal.DomainZonePatch{
		Spec: internal.SpecPatch{
			Records: map[string]any{
				subDomain: value,
			},
		},
	}

	_, err = d.client.PatchDomainZone(ctx, zone.Metadata.Name, patch)
	if err != nil {
		return fmt.Errorf("sotoon: remove TXT record: fqdn=%s: %w", info.EffectiveFQDN, err)
	}

	return nil
}

// Timeout returns the timeout and interval to use when checking for DNS propagation.
// Adjusting here to cope with spikes in propagation times.
func (d *DNSProvider) Timeout() (timeout, interval time.Duration) {
	return d.config.PropagationTimeout, d.config.PollingInterval
}

// Sequential All DNS challenges for this provider will be resolved sequentially.
// Returns the interval between each iteration.
func (d *DNSProvider) Sequential() time.Duration {
	return d.config.SequenceInterval
}

func (d *DNSProvider) findZone(ctx context.Context, fqdn string) (internal.DomainZone, error) {
	zones, err := d.client.ListDomainZones(ctx)
	if err != nil {
		return internal.DomainZone{}, fmt.Errorf("list domain zones: %w", err)
	}

	for name := range dns01.UnFqdnDomainsSeq(fqdn) {
		for _, zone := range zones {
			if strings.EqualFold(zone.Spec.Origin, name) {
				return zone, nil
			}
		}
	}

	return internal.DomainZone{}, fmt.Errorf("could not find zone for domain %q", fqdn)
}

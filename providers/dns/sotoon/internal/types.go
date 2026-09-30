package internal

import "fmt"

// DomainZoneList is a list of DomainZone resources.
type DomainZoneList struct {
	APIVersion string       `json:"apiVersion,omitempty"`
	Kind       string       `json:"kind,omitempty"`
	Items      []DomainZone `json:"items"`
}

// DomainZone is a Sotoon DomainZone resource.
type DomainZone struct {
	APIVersion string   `json:"apiVersion,omitempty"`
	Kind       string   `json:"kind,omitempty"`
	Metadata   Metadata `json:"metadata"`
	Spec       Spec     `json:"spec"`
}

// Metadata contains Kubernetes-style resource metadata.
type Metadata struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace,omitempty"`
}

// Spec contains the DomainZone specification.
type Spec struct {
	Email   string              `json:"email,omitempty"`
	Origin  string              `json:"origin"`
	Records map[string][]Record `json:"records,omitempty"`
}

// Record is a DNS record entry under a subdomain key.
type Record struct {
	TXT  string `json:"TXT,omitempty"`
	Type string `json:"type,omitempty"`
	TTL  int    `json:"ttl,omitempty"`
}

// DomainZonePatch is a JSON merge-patch payload for DomainZone.
type DomainZonePatch struct {
	Spec SpecPatch `json:"spec"`
}

// SpecPatch is the merge-patch for DomainZone.spec.
// Records values may be []Record or nil (to delete a subdomain key).
type SpecPatch struct {
	Records map[string]any `json:"records"`
}

// APIError is an API error response.
type APIError struct {
	Message string `json:"message"`
	Reason  string `json:"reason"`
	Code    int    `json:"code"`
}

func (a *APIError) Error() string {
	if a.Reason != "" {
		return fmt.Sprintf("%s: %s", a.Reason, a.Message)
	}

	return a.Message
}

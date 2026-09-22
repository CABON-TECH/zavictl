package models

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
)

// ResourceRef combines Kind and ID to uniquely identify a resource.
type ResourceRef string

// AuditMetadata holds tracking information.
type AuditMetadata struct {
	CreatedBy string `json:"created_by"`
	UpdatedBy string `json:"updated_by"`
}

// Resource contains the core fields shared by all resources.
// The field ResourceLabels is used instead of Labels so that
// the Labels() method can be defined to satisfy the interface.
type Resource struct {
	ID             string            `json:"id"`
	Kind           string            `json:"kind"`
	Name           string            `json:"name"`
	Description    string            `json:"description"`
	Owner          string            `json:"owner"`
	ResourceLabels map[string]string `json:"labels"`
	CreatedAt      time.Time         `json:"created_at"`
	UpdatedAt      time.Time         `json:"updated_at"`
	Version        int               `json:"version"`
	Status         string            `json:"status"`
	EnvironmentRef string            `json:"environment_ref,omitempty"`
	AuditMeta      AuditMetadata     `json:"audit_meta"`
}

// PersistedResource is the interface every kernel and provider type that is persisted must implement.
type PersistedResource interface {
	Identity() ResourceRef
	Validate() error
	Labels() map[string]string
}

// GenerateID generates a new sortable ULID string.
func GenerateID() string {
	return ulid.Make().String()
}

// Identity returns the ResourceRef for the resource.
func (r *Resource) Identity() ResourceRef {
	return ResourceRef(fmt.Sprintf("%s/%s", r.Kind, r.ID))
}

// Labels returns the resource's labels.
func (r *Resource) Labels() map[string]string {
	return r.ResourceLabels
}

// Validate performs structural and business-rule validation on the core fields.
func (r *Resource) Validate() error {
	if r.ID == "" {
		return errors.New("id is required")
	}
	if r.Kind == "" {
		return errors.New("kind is required")
	}
	if r.Name == "" {
		return errors.New("name is required")
	}
	if err := validateLabels(r.ResourceLabels); err != nil {
		return err
	}
	return nil
}

var (
	labelNameRegex   = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$`)
	labelPrefixRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$`)
	labelValueRegex  = regexp.MustCompile(`^([A-Za-z0-9][-A-Za-z0-9_.]*)?[A-Za-z0-9]$|^$`)
)

func validateLabels(labels map[string]string) error {
	for k, v := range labels {
		parts := strings.SplitN(k, "/", 2)
		var name string
		if len(parts) == 2 {
			prefix := parts[0]
			name = parts[1]
			if len(prefix) > 253 || !labelPrefixRegex.MatchString(prefix) {
				return fmt.Errorf("invalid label key prefix: %s", prefix)
			}
		} else {
			name = parts[0]
		}
		if len(name) > 63 || !labelNameRegex.MatchString(name) {
			return fmt.Errorf("invalid label key name: %s", name)
		}
		if len(v) > 63 || (len(v) > 0 && !labelValueRegex.MatchString(v)) {
			return fmt.Errorf("invalid label value: %s", v)
		}
	}
	return nil
}

package models

import (
	"fmt"
	"time"
)

// Project represents an organizational project.
type Project struct {
	Resource
}

func NewProject(name, description, owner string) *Project {
	now := time.Now().UTC()
	return &Project{
		Resource: Resource{
			ID:          GenerateID(),
			Kind:        "Project",
			Name:        name,
			Description: description,
			Owner:       owner,
			CreatedAt:   now,
			UpdatedAt:   now,
			Status:      "active",
		},
	}
}

// Environment represents a deployment target (e.g. staging, production).
type Environment struct {
	Resource
}

func NewEnvironment(name, description, owner string) *Environment {
	now := time.Now().UTC()
	return &Environment{
		Resource: Resource{
			ID:          GenerateID(),
			Kind:        "Environment",
			Name:        name,
			Description: description,
			Owner:       owner,
			CreatedAt:   now,
			UpdatedAt:   now,
			Status:      "active",
		},
	}
}

// Service represents a registered application or service.
type Service struct {
	Resource
	RepositoryURL string `json:"repository_url"`
}

func NewService(name, description, owner, repoURL, envRef string) *Service {
	now := time.Now().UTC()
	return &Service{
		Resource: Resource{
			ID:             GenerateID(),
			Kind:           "Service",
			Name:           name,
			Description:    description,
			Owner:          owner,
			EnvironmentRef: envRef,
			CreatedAt:      now,
			UpdatedAt:      now,
			Status:         "active",
		},
		RepositoryURL: repoURL,
	}
}

func (s *Service) Validate() error {
	if err := s.Resource.Validate(); err != nil {
		return err
	}
	if s.RepositoryURL == "" {
		return fmt.Errorf("repository URL is required")
	}
	return nil
}

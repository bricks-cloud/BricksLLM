package manager

import (
	"github.com/bricks-cloud/bricksllm/internal/policy"
)

type PoliciesStorage interface {
	CreatePolicy(p *policy.Policy) (*policy.Policy, error)
	UpdatePolicy(id string, p *policy.Policy) (*policy.Policy, error)
	GetPolicyById(id string) (*policy.Policy, error)
	GetPoliciesByTags(tags []string) ([]*policy.Policy, error)
}

type PolicyManager struct {
	Storage PoliciesStorage
}

func NewPolicyManager(s PoliciesStorage) *PolicyManager {
	return &PolicyManager{
		Storage: s,
	}
}

func (m *PolicyManager) CreatePolicy(p *policy.Policy) (*policy.Policy, error) {
	return m.Storage.CreatePolicy(p)
}

func (m *PolicyManager) UpdatePolicy(id string, p *policy.Policy) (*policy.Policy, error) {
	return m.Storage.UpdatePolicy(id, p)
}

func (m *PolicyManager) GetPoliciesByTags(tags []string) ([]*policy.Policy, error) {
	return m.Storage.GetPoliciesByTags(tags)
}

func (m *PolicyManager) GetPolicyById(id string) (*policy.Policy, error) {
	return m.Storage.GetPolicyById(id)
}

package manager

import (
	"time"

	"github.com/bricks-cloud/bricksllm/internal/policy"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type PoliciesStorage interface {
	CreatePolicy(p *policy.Policy) (*policy.Policy, error)
	UpdatePolicy(id string, p *policy.UpdatePolicy) (*policy.Policy, error)
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
	p.CreatedAt = time.Now().Unix()
	p.UpdatedAt = time.Now().Unix()
	p.Id = util.NewUuid()

	return m.Storage.CreatePolicy(p)
}

func (m *PolicyManager) UpdatePolicy(id string, p *policy.UpdatePolicy) (*policy.Policy, error) {
	p.UpdatedAt = time.Now().Unix()

	return m.Storage.UpdatePolicy(id, p)
}

func (m *PolicyManager) GetPoliciesByTags(tags []string) ([]*policy.Policy, error) {
	return m.Storage.GetPoliciesByTags(tags)
}

func (m *PolicyManager) GetPolicyById(id string) (*policy.Policy, error) {
	return m.Storage.GetPolicyById(id)
}

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

type PoliciesMemStorage interface {
	GetPolicy(id string) *policy.Policy
}

type PolicyManager struct {
	Storage PoliciesStorage
	Memdb   PoliciesMemStorage
}

func NewPolicyManager(s PoliciesStorage, memdb PoliciesMemStorage) *PolicyManager {
	return &PolicyManager{
		Storage: s,
		Memdb:   memdb,
	}
}

func (m *PolicyManager) CreatePolicy(p *policy.Policy) (*policy.Policy, error) {
	p.CreatedAt = time.Now().Unix()
	p.UpdatedAt = time.Now().Unix()
	p.Id = util.NewUuid()

	if p.Config == nil {
		p.Config = &policy.Config{}
	}

	if p.RegexConfig == nil {
		p.RegexConfig = &policy.RegexConfig{}
	}

	if p.CustomConfig == nil {
		p.CustomConfig = &policy.CustomConfig{}
	}

	return m.Storage.CreatePolicy(p)
}

func (m *PolicyManager) UpdatePolicy(id string, p *policy.UpdatePolicy) (*policy.Policy, error) {
	p.UpdatedAt = time.Now().Unix()

	return m.Storage.UpdatePolicy(id, p)
}

func (m *PolicyManager) GetPoliciesByTags(tags []string) ([]*policy.Policy, error) {
	return m.Storage.GetPoliciesByTags(tags)
}

func (m *PolicyManager) GetPolicyByIdFromMemdb(id string) *policy.Policy {
	return m.Memdb.GetPolicy(id)
}

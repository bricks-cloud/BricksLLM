package manager

import (
	"time"

	internal_errors "github.com/bricks-cloud/bricksllm/internal/errors"
	"github.com/bricks-cloud/bricksllm/internal/user"
	"github.com/bricks-cloud/bricksllm/internal/util"
)

type UserStorage interface {
	GetUsers(tags, keyIds, userIds []string, offset int, limit int) ([]*user.User, error)
	CreateUser(u *user.User) (*user.User, error)
	UpdateUser(id string, uu *user.UpdateUser) (*user.User, error)
}

type UserManager struct {
	us UserStorage
	ks Storage
}

func NewUserManager(us UserStorage, ks Storage) *UserManager {
	return &UserManager{
		us: us,
		ks: ks,
	}
}

func (m *UserManager) GetUsers(tags, keyIds, userIds []string, offset int, limit int) ([]*user.User, error) {
	return m.us.GetUsers(tags, keyIds, userIds, offset, limit)
}

func (m *UserManager) CreateUser(u *user.User) (*user.User, error) {
	u.CreatedAt = time.Now().Unix()
	u.UpdatedAt = time.Now().Unix()
	u.Id = util.NewUuid()

	if err := u.Validate(); err != nil {
		return nil, err
	}

	if len(u.KeyIds) != 0 {
		existing, err := m.ks.GetKeys(nil, u.KeyIds, "")
		if err != nil {
			return nil, err
		}

		if len(existing) == 0 {
			return nil, internal_errors.NewNotFoundError("keys are not found")
		}
	}

	return m.us.CreateUser(u)
}

func (m *UserManager) UpdateUser(id string, uu *user.UpdateUser) (*user.User, error) {
	uu.UpdatedAt = time.Now().Unix()

	if err := uu.Validate(); err != nil {
		return nil, err
	}

	if uu.KeyIds != nil && len(uu.KeyIds) != 0 {
		existing, err := m.ks.GetKeys(nil, uu.KeyIds, "")
		if err != nil {
			return nil, err
		}

		if len(existing) == 0 {
			return nil, internal_errors.NewNotFoundError("keys are not found")
		}
	}

	return m.us.UpdateUser(id, uu)
}

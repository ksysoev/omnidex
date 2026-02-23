package user

import (
	"testing"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	mock "github.com/stretchr/testify/mock"
)

func TestNew(t *testing.T) {
	daoMock := NewMockuserDAO(t)
	users := New(daoMock)

	assert.NotNil(t, users, "New() should return a non-nil UserRepo instance")
}

func TestUserRepo_CheckHealth(t *testing.T) {
	tests := []struct {
		setupMocks func(t *testing.T, dao *MockuserDAO)
		name       string // description of this test case
		wantErr    bool
	}{
		{
			name: "success",
			setupMocks: func(t *testing.T, dao *MockuserDAO) {
				t.Helper()
				dao.EXPECT().Ping(mock.Anything).Return(redis.NewStatusResult("", nil))
			},
			wantErr: false,
		},
		{
			name: "fail",
			setupMocks: func(t *testing.T, dao *MockuserDAO) {
				t.Helper()
				dao.EXPECT().Ping(mock.Anything).Return(redis.NewStatusResult("", assert.AnError))
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			daoMock := NewMockuserDAO(t)
			u := New(daoMock)

			tt.setupMocks(t, daoMock)

			err := u.CheckHealth(t.Context())

			if tt.wantErr {
				assert.Error(t, err, "CheckHealth() should return an error")
			} else {
				assert.NoError(t, err, "CheckHealth() should not return an error")
			}
		})
	}
}

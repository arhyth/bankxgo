package bankxgo_test

import (
	"testing"

	"github.com/arhyth/bankxgo"
	"github.com/arhyth/bankxgo/mocks"
	"github.com/bwmarrin/snowflake"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestNewService(t *testing.T) {
	t.Run("returns an error when a system account does not exist", func(tt *testing.T) {
		as := assert.New(tt)
		ctrl := gomock.NewController(t)
		repo := mocks.NewMockRepository(ctrl)
		sysAccts := map[string]snowflake.ID{
			"USD": snowflake.ParseInt64(7241301734201495552),
		}
		repo.EXPECT().
			GetAcct(sysAccts["USD"]).
			Return(nil, bankxgo.ErrNotFound{})
		_, err := bankxgo.NewService(repo, sysAccts)
		as.NotNil(err)
	})
}

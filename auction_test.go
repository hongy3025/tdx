package tdx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestResolveCodes_Empty(t *testing.T) {
	result := resolveCodes([]string{})
	assert.Nil(t, result)
}

func TestResolveCodes_All(t *testing.T) {
	result := resolveCodes([]string{"all"})
	assert.Nil(t, result)
}

func TestResolveCodes_Specific(t *testing.T) {
	result := resolveCodes([]string{"000001", "600000"})
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "sz000001", result[0])
	assert.Equal(t, "sh600000", result[1])
}

func TestResolveCodes_WithPrefix(t *testing.T) {
	result := resolveCodes([]string{"sz000001", "sh600000"})
	assert.Equal(t, 2, len(result))
	assert.Equal(t, "sz000001", result[0])
	assert.Equal(t, "sh600000", result[1])
}

func TestBatchGetCallAuction_EmptyCodes(t *testing.T) {
	cfg := AuctionConfig{
		Codes:    []string{},
		PoolSize: 10,
	}
	_, err := BatchGetCallAuction(nil, cfg)
	assert.Error(t, err)
}

func TestAuctionSnapshot_TableName(t *testing.T) {
	s := &AuctionSnapshot{}
	assert.Equal(t, "auction_snapshot", s.TableName())
}

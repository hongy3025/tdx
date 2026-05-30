package tdx

import (
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
)

// mockPool 实现 IPool 接口，用于测试
type mockPool struct {
	doFunc func(fn func(c *Client) error) error
}

func (m *mockPool) Get() (*Client, error)   { return nil, nil }
func (m *mockPool) Put(c *Client)            {}
func (m *mockPool) Go(fn func(c *Client)) error { return nil }
func (m *mockPool) Do(fn func(c *Client) error) error {
	if m.doFunc != nil {
		return m.doFunc(fn)
	}
	return fn(nil)
}

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

func TestBatchGetCallAuction_ConcurrentResults(t *testing.T) {
	var callCount int64
	pool := &mockPool{
		doFunc: func(fn func(c *Client) error) error {
			atomic.AddInt64(&callCount, 1)
			// 不调用 fn，模拟成功（无数据）
			return nil
		},
	}

	cfg := AuctionConfig{
		Codes:    []string{"sz000001", "sh600000", "sz000002"},
		PoolSize: 2,
	}

	results, err := BatchGetCallAuction(pool, cfg)
	assert.NoError(t, err)
	// 3 只股票各返回 1 条结果
	assert.Equal(t, 3, len(results))
	// 验证 pool.Do 被调用了 3 次（并发执行）
	assert.Equal(t, int64(3), atomic.LoadInt64(&callCount))

	// 验证每只股票都有结果
	codes := make(map[string]bool)
	for _, r := range results {
		codes[r.Code] = true
		assert.NoError(t, r.Error)
		assert.Nil(t, r.Auction) // 未调用 fn，无竞价数据
	}
	assert.True(t, codes["sz000001"])
	assert.True(t, codes["sh600000"])
	assert.True(t, codes["sz000002"])
}

func TestBatchGetCallAuction_PartialError(t *testing.T) {
	pool := &mockPool{
		doFunc: func(fn func(c *Client) error) error {
			// 模拟连接池获取失败
			return errors.New("连接池已耗尽")
		},
	}

	cfg := AuctionConfig{
		Codes:    []string{"sz000001", "sh600000"},
		PoolSize: 10,
	}

	results, err := BatchGetCallAuction(pool, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	for _, r := range results {
		assert.Error(t, r.Error)
		assert.Contains(t, r.Error.Error(), "连接池已耗尽")
	}
}

func TestBatchGetCallAuction_SuccessWithAuctionData(t *testing.T) {
	pool := &mockPool{
		doFunc: func(fn func(c *Client) error) error {
			// 不调用 fn，直接返回 nil（模拟成功但无数据）
			return nil
		},
	}

	cfg := AuctionConfig{
		Codes:    []string{"sz000001", "sh600000"},
		PoolSize: 10,
	}

	results, err := BatchGetCallAuction(pool, cfg)
	assert.NoError(t, err)
	assert.Equal(t, 2, len(results))
	for _, r := range results {
		assert.NoError(t, r.Error)
		assert.Nil(t, r.Auction) // 未调用 fn，无竞价数据
	}
}

func TestBatchGetCallAuction_ResultsIncludeAllCodes(t *testing.T) {
	var callCount int64
	pool := &mockPool{
		doFunc: func(fn func(c *Client) error) error {
			atomic.AddInt64(&callCount, 1)
			return nil
		},
	}

	codes := []string{"sz000001", "sh600000", "sz000002", "sh600036", "sz002230"}
	cfg := AuctionConfig{
		Codes:    codes,
		PoolSize: 3,
	}

	results, err := BatchGetCallAuction(pool, cfg)
	assert.NoError(t, err)
	assert.Equal(t, len(codes), len(results))
	assert.Equal(t, int64(len(codes)), atomic.LoadInt64(&callCount))

	// 验证结果中包含所有代码
	resultCodes := make(map[string]bool)
	for _, r := range results {
		resultCodes[r.Code] = true
	}
	for _, code := range codes {
		assert.True(t, resultCodes[code], "缺少 %s 的结果", code)
	}
}

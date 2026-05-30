package tdx

import (
	"fmt"
	"sync"
	"time"

	"github.com/injoyai/tdx/protocol"
	"github.com/panjf2000/ants/v2"
)

// AuctionSnapshot 竞价快照数据模型
type AuctionSnapshot struct {
	ID        int64     `json:"id" xorm:"pk autoincr"`
	Code      string    `json:"code" xorm:"index"`      // sz000001
	Name      string    `json:"name"`                    // 平安银行
	Time      time.Time `json:"time"`                    // 竞价时间
	Price     float64   `json:"price"`                   // 竞价价格
	Volume    int64     `json:"volume"`                  // 匹配量
	Unmatched int64     `json:"unmatched"`               // 未匹配量
	Flag      int8      `json:"flag"`                    // 1=买单, -1=卖单
	SnapTime  time.Time `json:"snap_time" xorm:"index"` // 快照采集时间
}

func (*AuctionSnapshot) TableName() string {
	return "auction_snapshot"
}

// AuctionConfig 竞价快照配置
type AuctionConfig struct {
	Codes    []string // 股票代码列表，空或 ["all"] 表示全市场
	PoolSize int      // ants 池大小，默认 50
}

// AuctionResult 单只股票竞价结果
type AuctionResult struct {
	Code    string                // 股票代码
	Name    string                // 股票名称
	Auction *protocol.CallAuction // 竞价数据
	Error   error                 // 错误信息
}

// resolveCodes 解析代码列表
func resolveCodes(codes []string) []string {
	if len(codes) == 0 || (len(codes) == 1 && codes[0] == "all") {
		if DefaultCodes != nil {
			return DefaultCodes.GetStockCodes()
		}
		return nil
	}
	// 补全市场前缀
	result := make([]string, 0, len(codes))
	for _, code := range codes {
		code = protocol.AddPrefix(code)
		result = append(result, code)
	}
	return result
}

// BatchGetCallAuction 批量获取竞价数据
func BatchGetCallAuction(pool IPool, cfg AuctionConfig) ([]*AuctionResult, error) {
	// 解析代码列表
	codes := resolveCodes(cfg.Codes)
	if len(codes) == 0 {
		return nil, fmt.Errorf("股票代码列表为空")
	}

	// 设置默认池大小
	if cfg.PoolSize <= 0 {
		cfg.PoolSize = 50
	}

	// 创建结果切片和互斥锁
	results := make([]*AuctionResult, 0, len(codes))
	var mu sync.Mutex
	var wg sync.WaitGroup

	// 创建 ants 池
	p, err := ants.NewPoolWithFunc(cfg.PoolSize, func(i interface{}) {
		defer wg.Done()

		code, ok := i.(string)
		if !ok {
			mu.Lock()
			results = append(results, &AuctionResult{
				Error: fmt.Errorf("ants 池收到非 string 类型参数: %T", i),
			})
			mu.Unlock()
			return
		}
		result := &AuctionResult{Code: code}

		// 从连接池获取客户端
		err := pool.Do(func(c *Client) error {
			// 获取股票名称
			if DefaultCodes != nil {
				m := DefaultCodes.Get(code)
				if m != nil {
					result.Name = m.Name
				}
			}

			// 获取竞价数据
			resp, err := c.GetCallAuction(code)
			if err != nil {
				result.Error = err
				return nil
			}
			if resp != nil && len(resp.List) > 0 {
				// 取最后一条竞价数据（最新）
				result.Auction = resp.List[len(resp.List)-1]
			}
			return nil
		})
		if err != nil {
			result.Error = err
		}

		// 收集结果
		mu.Lock()
		results = append(results, result)
		mu.Unlock()
	})
	if err != nil {
		return nil, fmt.Errorf("创建 ants 池失败: %w", err)
	}
	defer p.Release()

	// 提交所有任务
	snapTime := time.Now()
	for _, code := range codes {
		wg.Add(1)
		if err := p.Invoke(code); err != nil {
			wg.Done()
			results = append(results, &AuctionResult{
				Code:  code,
				Error: err,
			})
		}
	}

	// 等待所有任务完成
	wg.Wait()

	// 填充快照时间
	for _, r := range results {
		if r.Auction != nil {
			r.Auction.Time = snapTime
		}
	}

	return results, nil
}

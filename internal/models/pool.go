// 模型提供者 - 凭据池轮换/回退链
package models

import (
	"math/rand"
	"sync"
)

// Provider 模型提供者
type Provider struct {
	Name    string
	Model   string
	BaseURL string
	APIKey  string
	Weight  int    // 权重（越高越优先）
	Group   string // 分组：primary / fallback
}

// Pool 凭据池
type Pool struct {
	mu        sync.Mutex
	providers []Provider
	index     int
}

func NewPool() *Pool { return &Pool{} }

func (p *Pool) Add(pr Provider) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.providers = append(p.providers, pr)
}

// Next round-robin获取下一个提供者
func (p *Pool) Next() *Provider {
	p.mu.Lock()
	defer p.mu.Unlock()
	if len(p.providers) == 0 {
		return nil
	}
	pr := &p.providers[p.index]
	p.index = (p.index + 1) % len(p.providers)
	return pr
}

// Chain 回退链
type Chain struct {
	Primary []Provider
	Fallback []Provider
	current int
}

func NewChain(primary, fallback []Provider) *Chain {
	return &Chain{Primary: primary, Fallback: fallback}
}

// NextWithFallback 获取提供者，主链失败时自动降级
func (c *Chain) NextWithFallback() *Provider {
	if c.current < len(c.Primary) {
		pr := &c.Primary[c.current]
		c.current++
		return pr
	}
	if len(c.Fallback) > 0 {
		c.current = 0
		return &c.Fallback[rand.Intn(len(c.Fallback))]
	}
	return nil
}

// Reset 回退链到初始状态
func (c *Chain) Reset() { c.current = 0 }

package models

import (
	"testing"
)

func TestNewPool(t *testing.T) {
	p := NewPool()
	if p == nil {
		t.Fatal("expected non-nil pool")
	}
}

func TestPoolAddAndNext(t *testing.T) {
	p := NewPool()
	p.Add(Provider{Name: "deepseek", Model: "deepseek-chat"})
	p.Add(Provider{Name: "openai", Model: "gpt-4"})

	pr := p.Next()
	if pr == nil {
		t.Fatal("expected provider")
	}

	pr2 := p.Next()
	if pr2.Name == pr.Name {
		t.Error("expected round-robin to return different provider")
	}
}

func TestNewChain(t *testing.T) {
	primary := []Provider{{Name: "primary", Model: "deepseek"}}
	fallback := []Provider{{Name: "fallback", Model: "gpt-4"}}

	c := NewChain(primary, fallback)
	pr := c.NextWithFallback()
	if pr.Name != "primary" {
		t.Errorf("expected primary, got %s", pr.Name)
	}

	pr = c.NextWithFallback()
	if pr.Name != "fallback" {
		t.Errorf("expected fallback, got %s", pr.Name)
	}
}

func TestChainReset(t *testing.T) {
	c := NewChain([]Provider{{Name: "a"}}, nil)
	c.NextWithFallback()
	c.Reset()

	pr := c.NextWithFallback()
	if pr.Name != "a" {
		t.Error("expected reset to start from beginning")
	}
}

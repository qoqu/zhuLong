package i18n

import (
	"testing"
)

func TestNewBundle(t *testing.T) {
	b := New(LangZH)
	if b == nil {
		t.Fatal("expected non-nil bundle")
	}
	if b.defLang != LangZH {
		t.Errorf("expected default lang=zh, got %s", b.defLang)
	}
}

func TestRegister(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, map[string]string{"hello": "你好"})
	b.Register(LangEN, map[string]string{"hello": "Hello"})

	if b.T("hello", LangZH) != "你好" {
		t.Errorf("expected 你好, got %s", b.T("hello", LangZH))
	}
	if b.T("hello", LangEN) != "Hello" {
		t.Errorf("expected Hello, got %s", b.T("hello", LangEN))
	}
}

func TestFallback(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, map[string]string{"greeting": "你好"})

	// JA没有注册，应回退到ZH
	result := b.T("greeting", LangJA)
	if result != "你好" {
		t.Errorf("expected fallback to 你好, got %s", result)
	}
}

func TestKeyNotFound(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, map[string]string{"existing": "yes"})

	result := b.T("nonexistent", LangZH)
	if result != "nonexistent" {
		t.Errorf("expected key itself as fallback, got %s", result)
	}
}

func TestList(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, map[string]string{"a": "1"})
	b.Register(LangEN, map[string]string{"a": "2"})

	langs := b.List()
	if len(langs) != 2 {
		t.Errorf("expected 2 languages, got %d", len(langs))
	}
}

func TestHas(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, map[string]string{"a": "1"})

	if !b.Has(LangZH) {
		t.Error("expected Has(ZH) to be true")
	}
	if b.Has(LangEN) {
		t.Error("expected Has(EN) to be false")
	}
}

func TestDefaultTranslations(t *testing.T) {
	tl := DefaultTranslations()
	if tl["app.name"] != "烛龙" {
		t.Errorf("expected app.name=烛龙, got %s", tl["app.name"])
	}
	if len(tl) == 0 {
		t.Error("expected non-empty translations")
	}
}

func TestEnglishTranslations(t *testing.T) {
	tl := EnglishTranslations()
	if tl["app.name"] != "Zhulong" {
		t.Errorf("expected app.name=Zhulong, got %s", tl["app.name"])
	}
	if len(tl) == 0 {
		t.Error("expected non-empty translations")
	}
}

func TestBilingual(t *testing.T) {
	b := New(LangZH)
	b.Register(LangZH, DefaultTranslations())
	b.Register(LangEN, EnglishTranslations())

	if b.T("app.name", LangZH) != "烛龙" {
		t.Errorf("expected 烛龙, got %s", b.T("app.name", LangZH))
	}
	if b.T("app.name", LangEN) != "Zhulong" {
		t.Errorf("expected Zhulong, got %s", b.T("app.name", LangEN))
	}
	if b.T("agent.planning", LangZH) != "规划中..." {
		t.Errorf("expected 规划中..., got %s", b.T("agent.planning", LangZH))
	}
}

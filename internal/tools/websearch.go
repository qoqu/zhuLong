// 网络搜索工具
package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchTool 网络搜索工具
type WebSearchTool struct {
	apiKey string
	engine string // "duckduckgo" | "serpapi" | "bing"
}

// NewWebSearchTool creates a web search tool
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		engine: "duckduckgo",
	}
}

func (t *WebSearchTool) Name() string { return "web_search" }
func (t *WebSearchTool) Description() string {
	return "Search the web for information. Usage: web_search <query>"
}

// SearchResult represents a search result
type SearchResult struct {
	Title   string `json:"title"`
	URL     string `json:"url"`
	Snippet string `json:"snippet"`
}

func (t *WebSearchTool) Call(ctx context.Context, params map[string]interface{}) (string, error) {
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", fmt.Errorf("missing parameter: query")
	}

	// 使用 DuckDuckGo Instant Answer API（免费，无需 API Key）
	results, err := t.searchDuckDuckGo(ctx, query)
	if err != nil {
		// 回退：返回提示信息
		return fmt.Sprintf("搜索服务暂时不可用（%v）。请尝试其他方式获取信息。", err), nil
	}

	// 格式化结果
	var output strings.Builder
	output.WriteString(fmt.Sprintf("搜索结果: %s\n\n", query))
	for i, r := range results {
		if i >= 5 {
			break
		}
		output.WriteString(fmt.Sprintf("%d. %s\n   %s\n   %s\n\n", i+1, r.Title, r.URL, r.Snippet))
	}

	return output.String(), nil
}

// searchDuckDuckGo searches using DuckDuckGo
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string) ([]SearchResult, error) {
	// DuckDuckGo Instant Answer API
	apiURL := fmt.Sprintf("https://api.duckduckgo.com/?q=%s&format=json&no_html=1", url.QueryEscape(query))

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data struct {
		AbstractText string `json:"AbstractText"`
		AbstractURL  string `json:"AbstractURL"`
		Heading      string `json:"Heading"`
		RelatedTopics []struct {
			Text string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	var results []SearchResult

	// 添加摘要
	if data.AbstractText != "" {
		results = append(results, SearchResult{
			Title:   data.Heading,
			URL:     data.AbstractURL,
			Snippet: data.AbstractText,
		})
	}

	// 添加相关主题
	for _, topic := range data.RelatedTopics {
		if topic.Text != "" {
			results = append(results, SearchResult{
				Title:   topic.Text,
				URL:     topic.FirstURL,
				Snippet: topic.Text,
			})
		}
	}

	if len(results) == 0 {
		return []SearchResult{{Title: query, Snippet: "未找到直接结果，建议使用更具体的搜索词。"}}, nil
	}

	return results, nil
}

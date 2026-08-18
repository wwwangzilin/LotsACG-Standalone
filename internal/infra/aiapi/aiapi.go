// Package aiapi 提供一个 OpenAI 兼容的聊天补全客户端,
// 用于推荐系统中根据用户偏好 tag 自动生成关联的相似 tag。
package aiapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/goccy/go-json"
	"github.com/imroc/req/v3"
	"github.com/samber/oops"
	config "github.com/wwwangzilin/LotsACG-Standalone/internal/infra/config/runtimecfg"
	"github.com/wwwangzilin/LotsACG-Standalone/pkg/log"
)

// Client 是 OpenAI 兼容的 AI API 客户端。
type Client struct {
	cfg       config.AIAPIConfig
	reqClient *req.Client
}

// New 创建 AI API 客户端。优先使用 [aiapi] 配置, 若未启用则完全采用 [xpaiapi] 配置。
func New(aiapiCfg config.AIAPIConfig, xpCfg config.XPAIAPIConfig) *Client {
	cfg := aiapiCfg
	if !cfg.Enable {
		// 完全采用 xpaiapi 配置 (不使用 aiapi 的默认值, 避免发错端点)
		cfg = config.AIAPIConfig{
			Enable:  xpCfg.Enabled,
			BaseURL: xpCfg.BaseURL,
			APIKey:  xpCfg.APIKey,
			Model:   xpCfg.Model,
		}
	}
	c := req.C().
		SetLogger(log.Default()).
		SetTimeout(30 * time.Second).
		SetCommonRetryCount(2)
	if cfg.BaseURL != "" {
		c = c.SetBaseURL(cfg.BaseURL)
	}
	if cfg.APIKey != "" {
		c = c.SetCommonHeader("Authorization", "Bearer "+cfg.APIKey)
	}
	return &Client{
		cfg:       cfg,
		reqClient: c,
	}
}

func (c *Client) Enabled() bool {
	return c != nil && c.cfg.Enable && c.cfg.BaseURL != ""
}

type chatCompletionMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionRequest struct {
	Model       string                  `json:"model"`
	Messages    []chatCompletionMessage `json:"messages"`
	Temperature float64                 `json:"temperature"`
}

type chatCompletionResponse struct {
	Choices []struct {
		Message chatCompletionMessage `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error"`
}

// ExpandTags 根据给定的用户偏好 tags 生成 count 个相关联的 Pixiv 搜索 tag。
// 返回按重要性排序的 tag 列表; 若 AI 不可用或失败, 返回空切片(调用方应回退到原始 tags)。
func (c *Client) ExpandTags(ctx context.Context, tags []string, count int) ([]string, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if count <= 0 {
		count = 12
	}
	tagList := strings.Join(tags, ", ")
	prompt := fmt.Sprintf(`你是一名动漫插画(Pixiv)标签专家。以下是一位用户的偏好标签列表:
%s

请根据这些偏好标签, 生成 %d 个与之关联或相似的 Pixiv 搜索标签, 用于在 Pixiv 上搜索用户可能喜欢的新作品。
要求:
- 标签可以是同义词、常见搭配、风格、角色属性等
- 尽量使用 Pixiv 上常见的日文或英文标签, 也可以保留中文
- 不要包含 R-18 相关标签
- 只输出标签列表, 每个标签一行, 不要编号, 不要解释

生成的标签:`, tagList, count)

	req := chatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: "你是一个帮助生成动漫插画搜索标签的助手, 只输出标签列表, 不要输出其他内容。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.9,
	}

	var resp chatCompletionResponse
	httpResp, err := c.reqClient.R().
		SetContext(ctx).
		SetBody(req).
		SetSuccessResult(&resp).
		Post("/chat/completions")
	if err != nil {
		return nil, oops.Wrapf(err, "ai api request failed")
	}
	if httpResp.IsErrorState() {
		if resp.Error != nil {
			return nil, oops.Errorf("ai api error: %s", resp.Error.Message)
		}
		return nil, oops.Errorf("ai api http error: %d", httpResp.GetStatusCode())
	}
	if len(resp.Choices) == 0 {
		return nil, oops.New("ai api returned no choices")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, oops.New("ai api returned empty content")
	}
	return parseTagList(content), nil
}

// chat 执行一次聊天补全, 返回 assistant 文本内容。
func (c *Client) chat(ctx context.Context, system, user string, temperature float64) (string, error) {
	if !c.Enabled() {
		return "", oops.New("ai api not enabled")
	}
	req := chatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: system},
			{Role: "user", Content: user},
		},
		Temperature: temperature,
	}
	var resp chatCompletionResponse
	httpResp, err := c.reqClient.R().
		SetContext(ctx).
		SetBody(req).
		SetSuccessResult(&resp).
		Post("/chat/completions")
	if err != nil {
		return "", oops.Wrapf(err, "ai api request failed")
	}
	if httpResp.IsErrorState() {
		if resp.Error != nil {
			return "", oops.Errorf("ai api error: %s", resp.Error.Message)
		}
		return "", oops.Errorf("ai api http error: %d", httpResp.GetStatusCode())
	}
	if len(resp.Choices) == 0 {
		return "", oops.New("ai api returned no choices")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return "", oops.New("ai api returned empty content")
	}
	return content, nil
}

// GenerateDescription 根据作品标题与标签生成一段自然语言描述。
func (c *Client) GenerateDescription(ctx context.Context, title string, tags []string) (string, error) {
	if !c.Enabled() {
		return "", oops.New("ai api not enabled")
	}
	tagList := strings.Join(tags, ", ")
	if tagList == "" {
		tagList = "(无)"
	}
	prompt := `你是一个动漫插画描述专家。你的任务是将标题和标签转化为一段生动、有故事感的中文作品描述(80-150字)。
你可以灵活选择以下任意一种或混合多种形式：
- 画面描写：用有感染力的视觉语言描摹场景
- 角色独白/对话：以角色的口吻说出一句台词，展现性格和情绪
- 旁白叙事：用第三视角捕捉瞬间的情绪张力和故事感
- 碎片切片：捕捉一个微小动作、一次呼吸、一个眼神的重量

核心原则：
- 基于标签合理联想人物性格和情境，但不编造具体情节
- 语言要有温度和呼吸感，避免机械罗列
- 只输出最终描述正文，不附带任何其他内容`

	userPrompt := fmt.Sprintf(`请为以下作品生成描述：
标题：%s
标签：%s

提示：从标签中感知角色气质，让描述像故事的一个切面——可以是一句没说出口的话，可以是一个正在发生的瞬间，也可以让画面自带情绪。不要写"画面中"或"这幅作品"之类的元描述。`, title, tagList)
	content, err := c.chat(ctx, prompt, userPrompt, 0.9)
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(content), nil
}

// GenerateTags 根据作品标题与已有标签生成一组合理的 Pixiv 展示/搜索标签。
func (c *Client) GenerateTags(ctx context.Context, title string, existingTags []string) ([]string, error) {
	if !c.Enabled() {
		return nil, oops.New("ai api not enabled")
	}
	tagList := strings.Join(existingTags, ", ")
	if tagList == "" {
		tagList = "(无)"
	}
	prompt := fmt.Sprintf(`你是一名动漫插画(Pixiv)标签专家。根据作品标题与已有标签, 补充 8-15 个合理的 Pixiv 标签。
要求:
- 标签可以是角色属性、服装、场景、风格、题材等
- 尽量使用 Pixiv 上常见的日文或英文标签, 也可以保留中文
- 不要包含 R-18 相关标签
- 只输出标签列表, 每个标签一行, 不要编号, 不要解释

标题: %s
已有标签: %s

补充标签:`, title, tagList)
	content, err := c.chat(ctx, "你是一个帮助生成动漫插画标签的助手, 只输出标签列表, 不要输出其他内容。", prompt, 0.7)
	if err != nil {
		return nil, err
	}
	return parseTagList(content), nil
}

// parseTagList 解析 AI 返回的标签列表。兼容逗号分隔、换行分隔、以及带编号/引号/方括号的格式。
func parseTagList(content string) []string {
	// 去除可能的 markdown 代码块
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```text")
	content = strings.TrimPrefix(content, "```plaintext")
	content = strings.Trim(content, "`")

	// 尝试 JSON 数组
	if strings.HasPrefix(content, "[") && strings.HasSuffix(content, "]") {
		var arr []string
		if err := json.Unmarshal([]byte(content), &arr); err == nil {
			return cleanTags(arr)
		}
	}

	// 按换行或逗号分隔
	replacer := strings.NewReplacer("\r", "", "\n", ",", "，", ",", "、", ",")
	content = replacer.Replace(content)
	parts := strings.Split(content, ",")

	tags := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		p = strings.Trim(p, `"'*。-–—:：`)
		// 去掉 "1. " 之类的编号 (使用 rune 索引以安全处理多字节字符)
		runes := []rune(p)
		if len(runes) > 2 && (runes[1] == '.' || runes[1] == '、' || runes[1] == ')') {
			if runes[0] >= '0' && runes[0] <= '9' {
				p = string(runes[2:])
			}
		}
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		tags = append(tags, p)
	}
	return cleanTags(tags)
}

func cleanTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	out := make([]string, 0, len(tags))
	for _, t := range tags {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		if _, ok := seen[t]; ok {
			continue
		}
		seen[t] = struct{}{}
		out = append(out, t)
	}
	return out
}

// AICandidate 是待 AI 精排的候选作品。
type AICandidate struct {
	URL  string   `json:"url"`
	Tags []string `json:"tags"`
}

// ScoreArtworks 使用 LLM 对候选作品进行精排打分 (移植自 Pixiv-XP-Pusher AIScorer)。
// 返回 {url: score(0~1)} 映射。
func (c *Client) ScoreArtworks(ctx context.Context, prompt string, candidates []AICandidate) (map[string]float64, error) {
	if !c.Enabled() {
		return nil, nil
	}
	if len(candidates) == 0 {
		return nil, nil
	}

	req := chatCompletionRequest{
		Model: c.cfg.Model,
		Messages: []chatCompletionMessage{
			{Role: "system", Content: "你是推荐系统评分器, 只输出 JSON 数组, 不要输出其他内容。"},
			{Role: "user", Content: prompt},
		},
		Temperature: 0.3,
	}

	var resp chatCompletionResponse
	httpResp, err := c.reqClient.R().
		SetContext(ctx).
		SetBody(req).
		SetSuccessResult(&resp).
		Post("/chat/completions")
	if err != nil {
		return nil, oops.Wrapf(err, "ai api request failed")
	}
	if httpResp.IsErrorState() {
		if resp.Error != nil {
			return nil, oops.Errorf("ai api error: %s", resp.Error.Message)
		}
		return nil, oops.Errorf("ai api http error: %d", httpResp.GetStatusCode())
	}
	if len(resp.Choices) == 0 {
		return nil, oops.New("ai api returned no choices")
	}
	content := strings.TrimSpace(resp.Choices[0].Message.Content)
	if content == "" {
		return nil, oops.New("ai api returned empty content")
	}

	// 解析 JSON 数组 [{"url": "...", "score": 0.85}]
	content = strings.TrimSpace(content)
	content = strings.TrimPrefix(content, "```json")
	content = strings.TrimPrefix(content, "```")
	content = strings.TrimSuffix(content, "```")
	content = strings.Trim(content, "`")

	var items []struct {
		URL   string  `json:"url"`
		Score float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(content), &items); err != nil {
		return nil, oops.Wrapf(err, "failed to parse ai score response: %s", content)
	}
	result := make(map[string]float64, len(items))
	for _, item := range items {
		if item.URL == "" {
			continue
		}
		score := item.Score
		if score < 0 {
			score = 0
		}
		if score > 1 {
			score = 1
		}
		result[item.URL] = score
	}
	return result, nil
}

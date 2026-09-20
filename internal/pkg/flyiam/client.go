// Package flyiam 提供对 FlyIAM 服务间接口的调用（含自动获取 Casdoor 应用凭据）。
package flyiam

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// appCredentialsPath FlyIAM 返回 Casdoor 应用凭据的接口路径
const appCredentialsPath = "/api/casdoor/app-credentials"

// AppCredentials Casdoor 应用连接凭据
type AppCredentials struct {
	Endpoint     string `json:"endpoint"`
	Organization string `json:"organization"`
	Application  string `json:"application"`
	ClientId     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
}

// FetchAppCredentials 通过服务令牌从 FlyIAM 获取 Casdoor 应用凭据。
//
// 用途：本系统与 FlyIAM 复用同一 Casdoor，但无法自行获取应用凭据；
// 配置了 FlyIAM 地址 + 服务令牌后即可自动获取，免去手工填写 ClientID/Secret。
func FetchAppCredentials(ctx context.Context, endpoint, serviceToken string) (*AppCredentials, error) {
	endpoint = strings.TrimSpace(endpoint)
	serviceToken = strings.TrimSpace(serviceToken)
	if endpoint == "" || serviceToken == "" {
		return nil, fmt.Errorf("未配置 FlyIAM 地址或服务令牌")
	}

	url := strings.TrimRight(endpoint, "/") + appCredentialsPath
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("X-Service-Token", serviceToken)

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("请求 FlyIAM 失败: %w", err)
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("FlyIAM 返回 HTTP %d: %s", resp.StatusCode, snippet(raw))
	}

	var envelope struct {
		Code    int            `json:"code"`
		Message string         `json:"message"`
		Data    AppCredentials `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析响应失败（疑似非 FlyIAM 接口/返回了 HTML）: %s", snippet(raw))
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("FlyIAM 返回错误: %s", envelope.Message)
	}
	if envelope.Data.ClientId == "" || envelope.Data.ClientSecret == "" {
		return nil, fmt.Errorf("FlyIAM 未返回有效应用凭据")
	}
	return &envelope.Data, nil
}

func snippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

package userfield

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/iflyelf/consul_mgr/internal/model"
)

// flyiamExportPath FlyIAM 导出字段定义的接口路径
const flyiamExportPath = "/api/user-fields/export"

// SyncFromFlyIAM 从 FlyIAM 拉取字段定义并写入本地（按 field_key 幂等更新）。
//
// 目的：FlyIAM 从数据源同步用户时字段可能变化，consul_mgr 通过同步定义即可
// 自动跟上展示，无需修改代码。返回：新增数、更新数、总拉取数。
func (l *Logic) SyncFromFlyIAM(ctx context.Context, endpoint, serviceToken string) (int, int, int, error) {
	items, err := fetchFlyIAMFields(ctx, endpoint, serviceToken)
	if err != nil {
		return 0, 0, 0, err
	}

	// 现有定义（field_key -> 定义）
	existing, err := l.List(ctx)
	if err != nil {
		return 0, 0, 0, err
	}
	existingMap := make(map[string]*model.UserFieldDef, len(existing))
	for _, d := range existing {
		existingMap[d.FieldKey] = d
	}

	added, updated := 0, 0
	for i := range items {
		it := items[i]
		if err := l.UpsertFromSync(ctx, &it); err != nil {
			return added, updated, len(items), err
		}
		if old, ok := existingMap[it.FieldKey]; ok {
			if !sameField(old, &it) {
				updated++
			}
		} else {
			added++
		}
	}
	return added, updated, len(items), nil
}

// sameField 判断两个字段定义的展示属性是否一致
func sameField(a, b *model.UserFieldDef) bool {
	return a.Label == b.Label &&
		a.FieldType == b.FieldType &&
		a.Options == b.Options &&
		a.ShowInList == b.ShowInList &&
		a.ShowInForm == b.ShowInForm &&
		a.Editable == b.Editable &&
		a.SortOrder == b.SortOrder
}

// fetchFlyIAMFields 调用 FlyIAM 导出接口并解析字段定义
func fetchFlyIAMFields(ctx context.Context, endpoint, serviceToken string) ([]model.UserFieldDef, error) {
	url := strings.TrimRight(endpoint, "/") + flyiamExportPath
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
		Code    int                  `json:"code"`
		Message string               `json:"message"`
		Data    []model.UserFieldDef `json:"data"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return nil, fmt.Errorf("解析响应失败（疑似非 FlyIAM 接口/返回了 HTML）: %s", snippet(raw))
	}
	if envelope.Code != 0 {
		return nil, fmt.Errorf("FlyIAM 返回错误: %s", envelope.Message)
	}
	return envelope.Data, nil
}

func snippet(b []byte) string {
	s := strings.Join(strings.Fields(string(b)), " ")
	if len(s) > 200 {
		return s[:200] + "..."
	}
	return s
}

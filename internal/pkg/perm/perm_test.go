package perm

import "testing"

func TestPermissionAllows(t *testing.T) {
	cases := []struct {
		perms  []string
		action string
		want   bool
	}{
		{[]string{"read"}, "read", true},
		{[]string{"read"}, "write", false},
		{[]string{"*"}, "delete", true},
		{nil, "read", false},
	}
	for _, c := range cases {
		if got := permissionAllows(c.perms, c.action); got != c.want {
			t.Errorf("permissionAllows(%v,%s)=%v want %v", c.perms, c.action, got, c.want)
		}
	}
}

func TestServiceAllowed(t *testing.T) {
	cases := []struct {
		services []string
		target   string
		want     bool
	}{
		{nil, "a", false},                        // 空 = 无权限
		{[]string{}, "a", false},
		{[]string{"*"}, "a", true},               // 全部
		{[]string{"*"}, "", true},                // 组级 + 全部
		{[]string{"a", "b"}, "a", true},          // 精确命中
		{[]string{"a", "b"}, "c", false},         // 未授权
		{[]string{"a", "b"}, "", false},          // 组级操作需全部
	}
	for _, c := range cases {
		if got := serviceAllowed(c.services, c.target); got != c.want {
			t.Errorf("serviceAllowed(%v,%q)=%v want %v", c.services, c.target, got, c.want)
		}
	}
}

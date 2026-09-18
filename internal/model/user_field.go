package model

import "time"

// 字段类型常量
const (
	FieldTypeText     = "text"
	FieldTypeTextarea = "textarea"
	FieldTypeNumber   = "number"
	FieldTypeSelect   = "select"
	FieldTypeDate     = "date"
)

// UserFieldDef 用户字段定义（页面可管理，存储于数据库，零硬编码）。
//
// 与 FlyIAM 的 user_field_defs 结构对齐：字段名、显示名、类型、选项、
// 列表/表单可见性、可编辑、排序。用途：
//  1. 定义用户扩展字段的展示与编辑（值存于 Casdoor User.Properties，键为 FieldKey）；
//  2. 可从 FlyIAM 一键同步字段定义，数据源字段变化无需改代码。
type UserFieldDef struct {
	ID         int64     `db:"id" json:"id"`
	FieldKey   string    `db:"field_key" json:"fieldKey"`
	Label      string    `db:"label" json:"label"`
	FieldType  string    `db:"field_type" json:"fieldType"`
	Options    string    `db:"options" json:"options"`
	ShowInList bool      `db:"show_in_list" json:"showInList"`
	ShowInForm bool      `db:"show_in_form" json:"showInForm"`
	Editable   bool      `db:"editable" json:"editable"`
	Builtin    bool      `db:"builtin" json:"builtin"`
	SortOrder  int       `db:"sort_order" json:"sortOrder"`
	CreatedAt  time.Time `db:"created_at" json:"createdAt"`
	UpdatedAt  time.Time `db:"updated_at" json:"updatedAt"`
}

// BuiltinUserFields 返回内置人事字段定义（与 FlyIAM 保持一致）。
//
// 这些键与 FlyIAM 同步写入 Casdoor Properties 的键一致，因此用户列表中
// 即可直接展示由 FlyIAM 从数据源同步来的人事信息。
func BuiltinUserFields() []UserFieldDef {
	return []UserFieldDef{
		{FieldKey: "empCode", Label: "工号", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 10},
		{FieldKey: "deptNameLv0", Label: "零级部门", FieldType: FieldTypeText, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 20},
		{FieldKey: "deptNameLv1", Label: "一级部门", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 30},
		{FieldKey: "deptNameLv2", Label: "二级部门", FieldType: FieldTypeText, ShowInList: true, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 40},
		{FieldKey: "compileType", Label: "编制类型", FieldType: FieldTypeSelect, Options: `["正编","外包","实习"]`, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 50},
		{FieldKey: "superior", Label: "上级账号", FieldType: FieldTypeText, ShowInList: false, ShowInForm: true, Editable: true, Builtin: true, SortOrder: 60},
		{FieldKey: "deptIdLv0", Label: "零级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 70},
		{FieldKey: "deptIdLv1", Label: "一级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 80},
		{FieldKey: "deptIdLv2", Label: "二级部门ID", FieldType: FieldTypeText, ShowInList: false, ShowInForm: false, Editable: true, Builtin: true, SortOrder: 90},
	}
}

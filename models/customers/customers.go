package customers

import (
	"ferry/models/base"
)

type CustomersInfo struct {
	base.Model
	Name         string `gorm:"column:name; type:varchar(128)" json:"name" form:"name"`                          // 客户名
	Location     string `gorm:"column:location; type:text" json:"location" form:"location"`                      // 地址
	CreateUserID int    `gorm:"column:create_user_id; type:int(11)" json:"create_user_id" form:"create_user_id"` // 客户创建人

}

func (CustomersInfo) TableName() string {
	return "c_customers_info"
}

type ConnectsInfo struct {
	base.Model
	CustomersID int    `gorm:"column:customers_id; type:int(11)" json:"customers_id" form:"customers_id"` // 对应的客户
	Name        string `gorm:"column:name; type:varchar(128)" json:"name" form:"name"`                    // 联系人名
	Number      string `gorm:"column:number; type:varchar(32)" json:"number" form:"number"`               // 地址
}

func (ConnectsInfo) TableName() string {
	return "c_connects_info"
}

type ConnectsTag struct {
	base.Model
	ConnectsID int    `gorm:"column:connects_id; type:int(11)" json:"connects_id" form:"connects_id"` // 对应的客户
	Tag        string `gorm:"column:tag; type:varchar(128)" json:"tag" form:"tag"`                    // 联系人名
}

func (ConnectsTag) TableName() string {
	return "c_connects_tag"
}

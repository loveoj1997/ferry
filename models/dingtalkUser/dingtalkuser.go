package dingtalkUser

import (
	"ferry/global/orm"
)

const DepartmentList = "https://oapi.dingtalk.com/topapi/v2/department/listsub"

type DeptBaseResponse struct {
	DeptID          int64  `json:"dept_id"`
	Name            string `json:"name"`
	ParentID        int64  `json:"parent_id"`
	CreateDeptGroup bool   `json:"create_dept_group"`
	AutoAddUser     bool   `json:"auto_add_user"`
}

type GetDepartmentListRsp struct {
	RequestID string             `json:"request_id"`
	ErrCode   string             `json:"errcode"`
	ErrMsg    string             `json:"errmsg"`
	Result    []DeptBaseResponse `json:"result"`
}

const ListDeptUserIDs = "https://oapi.dingtalk.com/topapi/user/listid"

type ListDeptUserIDsReq struct {
	DeptID int64 `json:"dept_id"`
}

type ListDeptUserIDsRsp struct {
	Errcode    float64 `json:"errcode"`
	Errmsg     string  `json:"errmsg"`
	Result     Result  `json:"result"`
	Request_id string  `json:"request_id"`
}

type Result struct {
	Userid_list []string `json:"userid_list"`
}

const (
	GetUserIDByUnionID = "https://oapi.dingtalk.com/topapi/user/getbyunionid"
	GetUserInfoByDept  = "https://oapi.dingtalk.com/topapi/v2/user/list"
)

type GetUserIDByUnionIDReq struct {
	Unionid string `json:"unionid"`
}

type GetUserListByDeptReq struct {
	Cursor             int  `json:"cursor"`
	ContainAccessLimit bool `json:"contain_access_limit"`
	Size               int  `json:"size"`
	DeptID             int  `json:"dept_id"`
}

type UserInfo struct {
	Unionid string `json:"unionid"`
	Userid  string `json:"userid"`
	Email   string `json:"email"`
	Mobile  string `json:"mobile"`
	Name    string `json:"name"`
}
type GetUserListByDeptResp struct {
	Errcode string `json:"errcode"`
	Result  struct {
		NextCursor string     `json:"next_cursor"`
		HasMore    string     `json:"has_more"`
		List       []UserInfo `json:"list"`
	} `json:"result"`
	Errmsg string `json:"errmsg"`
}

type GetUserIDByUnionIDRsp struct {
	Result struct {
		Userid string `json:"userid"`
	} `json:"result"`
}

func (e *UserInfos) UpsertUser(id int) (userInfo *UserInfos, err error) {
	var currentUser *UserInfos
	if err = orm.Eloquent.Table(e.TableName()).Where("unionid = ? and userid = ?", userInfo.Unionid, userInfo.Userid).First(&currentUser).Error; err != nil {
		return
	}

	if currentUser != nil && currentUser.ID > 0 { // 更新userinfo内容

		// TODO : 如果这里会有产生空值的可能，就改成updates
		orm.Eloquent.Table(e.TableName()).Model(&UserInfos{}).Where("_id = ?", currentUser.ID).Save(currentUser)
		userInfo.ID = currentUser.ID
	} else {
		orm.Eloquent.Table(e.TableName()).Model(&UserInfos{}).Create(&userInfo)
	}

	return
}
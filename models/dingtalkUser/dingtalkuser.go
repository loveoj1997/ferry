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

const GetUserIDByUnionID = "https://oapi.dingtalk.com/topapi/user/getbyunionid"

type GetUserIDByUnionIDReq struct {
	Unionid string `json:"unionid"`
}

type GetUserIDByUnionIDRsp struct {
	Errcode string `json:"errcode"`
	Errmsg  string `json:"errmsg"`
	Result  struct {
		ContactType int    `json:"contact_type"`
		Userid      string `json:"userid"`
	} `json:"result"`
	RequestID string `json:"request_id"`
}

const GetUserDetails = "https://oapi.dingtalk.com/topapi/v2/user/get"

type UserInfoDetailsRsp struct {
	Errcode string                `json:"errcode"`
	Result  UserInfoDetailsResult `json:"result"`
	Errmsg  string                `json:"errmsg"`
}
type RoleList struct {
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	ID        string `json:"id"`
}

type LeaderInDept struct {
	Leader string `json:"leader"`
	DeptID string `json:"dept_id"`
}
type UnionEmpMapList struct {
	Userid string `json:"userid"`
	CorpID string `json:"corp_id"`
}
type UnionEmpExt struct {
	UnionEmpMapList UnionEmpMapList `json:"union_emp_map_list"`
	Userid          string          `json:"userid"`
	CorpID          string          `json:"corp_id"`
}
type UserInfoDetailsResult struct {
	Unionid          string       `json:"unionid"`
	Boss             bool         `json:"boss"`
	RoleList         RoleList     `json:"role_list"`
	ExclusiveAccount bool         `json:"exclusive_account"`
	ManagerUserid    string       `json:"manager_userid"`
	Admin            bool         `json:"admin"`
	Title            string       `json:"title"`
	Userid           string       `json:"userid"`
	DeptIDList       []int64      `json:"dept_id_list"`
	JobNumber        string       `json:"job_number"`
	Email            string       `json:"email"`
	LeaderInDept     LeaderInDept `json:"leader_in_dept"`
	Mobile           string       `json:"mobile"`
	Active           string       `json:"active"`
	OrgEmail         string       `json:"org_email"`
	Avatar           string       `json:"avatar"`
	Senior           bool         `json:"senior"`
	Name             string       `json:"name"`
	StateCode        string       `json:"state_code"`
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

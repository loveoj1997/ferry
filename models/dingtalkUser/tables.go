package dingtalkUser

type Roles struct {
	ID        int64  `gorm:"_id"`
	GroupName string `json:"group_name"`
	Name      string `json:"name"`
	RoleID    int64  `json:"role_id"`
}

func (Roles) TableName() string {
	return "dingtalk_roles"
}

type Depts struct {
	ID           int64  `gorm:"_id"`
	DeptName     string `json:"dept_name"`
	DeptID       int64  `json:"dept_id"`
	ParentDeptID int64  `json:"parent_dept_id"`
}

func (Depts) TableName() string {
	return "dingtalk_depts"
}

type UserInfos struct {
	ID               int64  `gorm:"_id"`
	Unionid          string `json:"unionid"`
	Boss             bool   `json:"boss"`
	ExclusiveAccount bool   `json:"exclusive_account"`
	ManagerUserid    string `json:"manager_userid"`
	Admin            bool   `json:"admin"`
	Title            string `json:"title"`
	Userid           string `json:"userid"`
	JobNumber        string `json:"job_number"`
	Email            string `json:"email"`
	Mobile           string `json:"mobile"`
	Active           string `json:"active"`
	OrgEmail         string `json:"org_email"`
	Avatar           string `json:"avatar"`
	Senior           bool   `json:"senior"`
	Name             string `json:"name"`
	StateCode        string `json:"state_code"`
}

func (UserInfos) TableName() string {
	return "dingtalk_users"
}

type UserRole struct {
	ID      int64 `gorm:"_id"`
	UnionID int64 `json:"unionid"`
	RoleID  int64 `json:"role_id"`
}

func (UserRole) TableName() string {
	return "user_role"
}

type UserDept struct {
	ID       int64 `gorm:"_id"`
	UnionID  int64 `json:"unionid"`
	DeptID   int64 `json:"dept_id"`
	IsLeader bool  `gorm:"is_leader"`
}

func (UserDept) TableName() string {
	return "user_dept"
}

package handler

import (
	"encoding/json"
	"errors"
	"ferry/global/orm"
	"ferry/models/dingtalkUser"
	"ferry/models/system"
	jwt "ferry/pkg/jwtauth"
	ldap1 "ferry/pkg/ldap"
	"ferry/pkg/logger"
	"ferry/pkg/settings"
	"ferry/tools"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	"github.com/go-ldap/ldap/v3"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"github.com/mssola/user_agent"

	openapi "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	dingtalkcontact "github.com/alibabacloud-go/dingtalk/contact_1_0"
	dingtalkoauth "github.com/alibabacloud-go/dingtalk/oauth2_1_0"
	teaUtil "github.com/alibabacloud-go/tea-utils/v2/service"
)

var store = base64Captcha.DefaultMemStore

func PayloadFunc(data interface{}) jwt.MapClaims {
	if v, ok := data.(map[string]interface{}); ok {
		u, _ := v["user"].(system.SysUser)
		r, _ := v["role"].(system.SysRole)
		return jwt.MapClaims{
			jwt.IdentityKey: u.UserId,
			jwt.RoleIdKey:   r.RoleId,
			jwt.RoleKey:     r.RoleKey,
			jwt.NiceKey:     u.Username,
			jwt.RoleNameKey: r.RoleName,
		}
	}
	return jwt.MapClaims{}
}

func IdentityHandler(c *gin.Context) interface{} {
	claims := jwt.ExtractClaims(c)
	return map[string]interface{}{
		"IdentityKey": claims["identity"],
		"UserName":    claims["nice"],
		"RoleKey":     claims["rolekey"],
		"UserId":      claims["identity"],
		"RoleIds":     claims["roleid"],
	}
}

// @Summary 登陆
// @Description 获取token
// LoginHandler can be used by clients to get a jwt token.
// Payload needs to be json in the form of {"username": "USERNAME", "password": "PASSWORD"}.
// Reply will be of the form {"token": "TOKEN"}.
// @Accept  application/json
// @Product application/json
// @Param username body system.Login  true "Add account"
// @Success 200 {string} string "{"code": 200, "expire": "2019-08-07T12:45:48+08:00", "token": ".eyJleHAiOjE1NjUxNTMxNDgsImlkIjoiYWRtaW4iLCJvcmlnX2lhdCI6MTU2NTE0OTU0OH0.-zvzHvbg0A" }"
// @Router /login [post]
func Authenticator(c *gin.Context) (interface{}, error) {
	var (
		err           error
		loginVal      system.Login
		loginLog      system.LoginLog
		roleValue     system.SysRole
		authUserCount int
		addUserInfo   system.SysUser
		ldapUserInfo  *ldap.Entry
		isVerifyCode  interface{}
	)

	ua := user_agent.New(c.Request.UserAgent())
	loginLog.Ipaddr = c.ClientIP()
	location := tools.GetLocation(c.ClientIP())
	loginLog.LoginLocation = location
	loginLog.LoginTime = tools.GetCurrntTime()
	loginLog.Status = "0"
	loginLog.Remark = c.Request.UserAgent()
	browserName, browserVersion := ua.Browser()
	loginLog.Browser = browserName + " " + browserVersion
	loginLog.Os = ua.OS()
	loginLog.Msg = "登录成功"
	loginLog.Platform = ua.Platform()

	// 获取前端过来的数据
	if err := c.ShouldBind(&loginVal); err != nil {
		loginLog.Status = "1"
		loginLog.Msg = "数据解析失败"
		loginLog.Username = loginVal.Username
		_, _ = loginLog.Create()
		return nil, jwt.ErrMissingLoginValues
	}
	loginLog.Username = loginVal.Username

	// 查询设置 is_verify_code
	isVerifyCode, err = settings.GetContentByKey(1, "is_verify_code")
	if err != nil {
		return nil, errors.New("获取是否需要验证码校验失败")
	}

	if isVerifyCode != nil && isVerifyCode.(bool) {
		// 校验验证码
		if !store.Verify(loginVal.UUID, loginVal.Code, true) {
			loginLog.Status = "1"
			loginLog.Msg = "验证码错误"
			_, _ = loginLog.Create()
			return nil, jwt.ErrInvalidVerificationode
		}
	}

	// ldap 验证
	if loginVal.LoginType == 1 {
		// ldap登陆
		ldapUserInfo, err = ldap1.LdapLogin(loginVal.Username, loginVal.Password)
		if err != nil {
			return nil, err
		}
		// 2. 将ldap用户信息写入到用户数据表中
		err = orm.Eloquent.Model(&system.SysUser{}).
			Where("username = ?", loginVal.Username).
			Count(&authUserCount).Error
		if err != nil {
			return nil, errors.New(fmt.Sprintf("查询用户失败，%v", err))
		}
		addUserInfo, err = ldap1.LdapFieldsMap(ldapUserInfo)
		if err != nil {
			return nil, fmt.Errorf("ldap映射本地字段失败，%v", err.Error())
		}
		if authUserCount == 0 {
			addUserInfo.Username = loginVal.Username
			// 获取默认权限ID
			err = orm.Eloquent.Model(&system.SysRole{}).Where("role_key = 'common'").Find(&roleValue).Error
			if err != nil {
				return nil, errors.New(fmt.Sprintf("查询角色失败，%v", err))
			}
			addUserInfo.RoleId = roleValue.RoleId // 绑定通用角色
			addUserInfo.Status = "0"
			addUserInfo.CreatedAt = time.Now()
			addUserInfo.UpdatedAt = time.Now()
			if addUserInfo.Sex == "" {
				addUserInfo.Sex = "0"
			}
			err = orm.Eloquent.Create(&addUserInfo).Error
			if err != nil {
				return nil, errors.New(fmt.Sprintf("创建本地用户失败，%v", err))
			}
		}
	}

	user, role, e := loginVal.GetUser()
	if e == nil {
		_, _ = loginLog.Create()

		if user.Status == "1" {
			return nil, errors.New("用户已被禁用。")
		}

		return map[string]interface{}{"user": user, "role": role}, nil
	} else {
		loginLog.Status = "1"
		loginLog.Msg = "登录失败"
		_, _ = loginLog.Create()
		logger.Info(e.Error())
	}

	return nil, jwt.ErrFailedAuthentication
}

func CreateOauthClient() (_result *dingtalkoauth.Client, _err error) {
	config := &openapi.Config{}
	config.SetProtocol("https")
	config.SetRegionId("central")
	config.SetConnectTimeout(30)
	_result = &dingtalkoauth.Client{}
	_result, _err = dingtalkoauth.NewClient(config)
	return _result, _err
}

func CreateContactClient() (_result *dingtalkcontact.Client, _err error) {
	config := &openapi.Config{}
	config.Protocol = tea.String("https")
	config.RegionId = tea.String("central")
	_result = &dingtalkcontact.Client{}
	_result, _err = dingtalkcontact.NewClient(config)
	return _result, _err
}

func GetAccessToken(appKey, appSecret string) (accessToken *string, err error) {
	getAccessTokenRequest := &dingtalkoauth.GetAccessTokenRequest{
		AppKey:    tea.String(appKey),
		AppSecret: tea.String(appSecret),
	}

	client, _ := CreateOauthClient()
	resp, err := client.GetAccessToken(getAccessTokenRequest)
	if err != nil {
		return nil, err
	}
	if resp.Body.AccessToken != nil {
		return resp.Body.AccessToken, err
	} else {
		return nil, nil
	}
}

func GetUserAccessToken(appKey, appSecret, code string) (respBody *dingtalkoauth.GetUserTokenResponseBody, err error) {
	logger.Info("app key is :", appKey)
	logger.Info("app sec is :", appSecret)
	logger.Info("code is :", code)
	getUserTokenReq := &dingtalkoauth.GetUserTokenRequest{
		ClientId:     tea.String(appKey),
		ClientSecret: tea.String(appSecret),
		Code:         tea.String(code),
		GrantType:    tea.String("authorization_code"),
	}

	client, _ := CreateOauthClient()
	resp, err := client.GetUserToken(getUserTokenReq)
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		return resp.Body, err
	} else {
		return nil, nil
	}
}

func GetUserInfoByToken(userAccessToken *string) (respBody *dingtalkcontact.GetUserResponseBody, err error) {
	logger.Info("user token is :", *userAccessToken)
	getUserHeaders := &dingtalkcontact.GetUserHeaders{}
	getUserHeaders.XAcsDingtalkAccessToken = userAccessToken
	client, _ := CreateContactClient()

	resp, err := client.GetUserWithOptions(tea.String("me"), getUserHeaders, &teaUtil.RuntimeOptions{})
	if err != nil {
		return nil, err
	}
	if resp.Body != nil {
		return resp.Body, err
	} else {
		return nil, nil
	}
}

func GetUserIDByUnionID(appAccessToken, unionID string) (userID *string, err error) {
	logger.Info("unionID  is :", unionID)

	getByUnionUrl := dingtalkUser.GetUserIDByUnionID + "?access_token=" + appAccessToken

	payload := &dingtalkUser.GetUserIDByUnionIDReq{
		Unionid: unionID,
	}
	payloadJson, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", getByUnionUrl, strings.NewReader(string(payloadJson)))

	req.Header.Add("Content-Type", "application/json")

	response, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}

	defer response.Body.Close()
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return nil, err
	}

	resp := new(dingtalkUser.GetUserIDByUnionIDRsp)
	err = json.Unmarshal(body, resp)
	if err != nil {
		return nil, err
	}

	return &resp.Result.Userid, nil
}

func GetUserByDept(appAccessToken string) (userInfoList []dingtalkUser.DingtalkUserInfo, err error) {
	getByUnionUrl := dingtalkUser.GetUserInfoByDept + "?access_token=" + appAccessToken

	payload := &dingtalkUser.GetUserListByDeptReq{
		Cursor:             0,
		Size:               100,
		ContainAccessLimit: true,
	}

	userInfoList = make([]dingtalkUser.DingtalkUserInfo, 0)

	for _, d := range deptList {
		payload.DeptID = d
		payloadJson, _ := json.Marshal(payload)

		req, _ := http.NewRequest("POST", getByUnionUrl, strings.NewReader(string(payloadJson)))

		req.Header.Add("Content-Type", "application/json")

		response, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}

		defer response.Body.Close()
		body, err := io.ReadAll(response.Body)
		if err != nil {
			return nil, err
		}

		resp := new(dingtalkUser.GetUserListByDeptResp)
		err = json.Unmarshal(body, resp)
		if err != nil {
			return nil, err
		}

		for _, u := range resp.Result.List {
			userInfoList = append(userInfoList, u)
		}
	}

	return userInfoList, nil
}

func GetUserDal(appKey, appSecret, code string) (userDetail *dingtalkUser.UserInfos, err error) {
	userTokenBody, err := GetUserAccessToken(appKey, appSecret, code)
	if err != nil {
		return nil, err
	}
	userInfo, err := GetUserInfoByToken(userTokenBody.AccessToken)
	if err != nil {
		return nil, err
	}

	appAccessToken, err := GetAccessToken(appKey, appSecret)
	if err != nil {
		return nil, err
	}

	userID, err := GetUserIDByUnionID(*appAccessToken, *userInfo.UnionId)
	if err != nil {
		return nil, err
	}

	userDetail = &dingtalkUser.UserInfos{}

	if userInfo.UnionId != nil {
		userDetail.Unionid = *userInfo.UnionId
	}
	if userID != nil {
		userDetail.Userid = *userID
	}
	if userInfo.Email != nil {
		userDetail.Email = *userInfo.Email
	}
	if userInfo.UnionId != nil {
		userDetail.Mobile = *userInfo.Mobile
	}
	if userInfo.AvatarUrl != nil {
		userDetail.Avatar = *userInfo.AvatarUrl
	}
	if userInfo.Nick != nil {
		userDetail.Name = *userInfo.Nick
	}

	return userDetail, nil
}

func FetchUser(appKey, appSecret string) ([]dingtalkUser.DingtalkUserInfo, error) {
	appAccessToken, err := GetAccessToken(appKey, appSecret)
	if err != nil {
		return nil, err
	}

	userInfos, err := GetUserByDept(*appAccessToken)
	if err != nil {
		return nil, err
	}

	usersDetail := make([]dingtalkUser.DingtalkUserInfo, 0)
	for _, userInfo := range userInfos {
		var userDetail dingtalkUser.DingtalkUserInfo

		if userInfo.Unionid != "" {
			userDetail.Unionid = userInfo.Unionid
		}
		if userInfo.Userid != "" {
			userDetail.Userid = userInfo.Userid
		}
		if userInfo.Email != "" {
			userDetail.Email = userInfo.Email
		}
		if userInfo.Name != "" {
			userDetail.Name = userInfo.Name
		}
		if userInfo.Mobile != "" {
			userDetail.Mobile = userInfo.Mobile
		}

		usersDetail = append(usersDetail, userDetail)
	}

	return usersDetail, nil
}

func DingtalkAuthenticator(c *gin.Context) (interface{}, error) {
	var (
		err      error
		loginVal system.DingtalkLogin
		loginLog system.LoginLog
		sysUser  system.SysUser
		sysRole  system.SysRole
	)

	ua := user_agent.New(c.Request.UserAgent())
	loginLog.Ipaddr = c.ClientIP()
	location := tools.GetLocation(c.ClientIP())
	loginLog.LoginLocation = location
	loginLog.LoginTime = tools.GetCurrntTime()
	loginLog.Status = "0"
	loginLog.Remark = c.Request.UserAgent()
	browserName, browserVersion := ua.Browser()
	loginLog.Browser = browserName + " " + browserVersion
	loginLog.Os = ua.OS()
	loginLog.Msg = "登录成功"
	loginLog.Platform = ua.Platform()

	// 获取前端过来的数据
	if err := c.ShouldBindQuery(&loginVal); err != nil {
		loginLog.Status = "1"
		loginLog.Msg = "authCode数据解析失败"
		_, _ = loginLog.Create()
		return nil, jwt.ErrMissingLoginValues
	}

	userInfo, err := GetUserDal("dingqqx81mjesm5lqmgx", "HE-F4i46VLTunrahg_6jj48POm3MFG5GrSrjIshaM-uBh1Pknd7_Ua_KhjGkgE3X", loginVal.AuthCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "internal server error ," + err.Error(),
		})
		return nil, err
	}
	if userInfo == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "no user info found",
		})
		return nil, err
	}

	loginLog.Username = userInfo.Name

	sysUser, sysRole, err = sysUser.UpsertDingtalkUser(userInfo)
	if err == nil {
		_, _ = loginLog.Create()

		return map[string]interface{}{"user": sysUser, "role": sysRole}, nil
	} else {
		loginLog.Status = "1"
		loginLog.Msg = "登录失败"
		_, _ = loginLog.Create()
		logger.Info(err.Error())
	}

	return nil, jwt.ErrFailedAuthentication
}

const (
	botKey = "dingqqx81mjesm5lqmgx"
	botSec = "HE-F4i46VLTunrahg_6jj48POm3MFG5GrSrjIshaM-uBh1Pknd7_Ua_KhjGkgE3X"

	// 钉钉部门ID，常数，部门结构未发生变化的情况下勿动。
	rootDept       = 1
	commerceDept   = 7825695   // 商务部
	deputyGM       = 7826896   // 副总
	designDept     = 7827632   // 设计部
	systemDept     = 7832641   // 系统集成部
	financeDept    = 7840056   // 财务部
	officeDept     = 7843002   // 办公室
	generalManager = 8085621   // 总经理
	operationDept  = 8102954   // 运行维护
	szjyOP         = 12416580  // 深州监狱
	tcjcOP         = 354309639 // 桃城区检察院
	hszyOP         = 458005469 // 衡水中院
	wqOP           = 458286033 // 武强
	jxOP           = 458567160 // 景县
	fcOP           = 458852104 // 阜城
	ryOP           = 459026129 // 饶阳
	zqOP           = 499188422 // 枣强
	hsgaOP         = 503979491 // 衡水公安
	apOP           = 657239207 // 安平
	tcfyOP         = 908491341 // 桃城区法院
	hsswOP         = 908793080 // 衡水市委
	whgOP          = 908845086 // 衡水文化宫
	hsjcOP         = 913175227 // 衡水市检察院
	digitizingDept = 33582037  // 数字化部
	digitTempDept  = 408433229 // 数字化部临时
	businessDept   = 68840530  // 业务部
	chiefEngineer  = 95484111  // 总工程师
	director       = 661205352 // 总监理工程师
	deputyChief    = 664673055 // 常务副总经理
)

var deptList = [...]int{rootDept, commerceDept, deputyGM, designDept, systemDept, financeDept, officeDept,
	generalManager, operationDept, szjyOP, tcjcOP, hszyOP, wqOP, jxOP, fcOP, ryOP, zqOP, hsgaOP, apOP,
	tcfyOP, hsswOP, whgOP, hsjcOP, digitizingDept, digitTempDept, businessDept,
	chiefEngineer, director, deputyChief}

const (
	nocobaseAuth       = "http://localhost:13000/api/auth:signIn"
	nocobaseUserCreate = "http://localhost:13000/api/users:create"
	nocobaseUserList   = "http://localhost:13000/api/users:list"
)

type nocobaseAuthReq struct {
	Account  string `json:"account"`
	Password string `json:"password"`
}

type nocobaseAuthResp struct {
	Data struct {
		Token string `json:"token"`
	} `json:"data"`
}

type nocobaseUserInfo struct {
	Username string `json:"username"`
	Email    string `json:"email"`
	Phone    string `json:"phone"`
	Nickname string `json:"nickname"`
	Userid   string `json:"userid"`
	Name     string `json:"name"`
	Unionid  string `json:"unionid"`
	Password string `json:"password"`
}

//type nocobaseUserListResp struct {
//	Data []nocobaseUserInfo `json:"data"`
//	Meta Meta               `json:"meta"`
//}
//

//type Meta struct {
//	Count     int `json:"count"`
//	Page      int `json:"page"`
//	PageSize  int `json:"pageSize"`
//	TotalPage int `json:"totalPage"`
//}

func DingtalkCreateUsers(c *gin.Context) error {
	var (
		err error
	)

	userInfos, err := FetchUser(botKey, botSec)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"msg": "internal server error ," + err.Error(),
		})
		return err
	}
	if userInfos == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"msg": "no user info found",
		})
		return err
	}

	nocobaseAuthPayload := &nocobaseAuthReq{
		Account:  "admin@nocobase.com",
		Password: "admin123",
	}
	nocobaseAuthPayloadJson, _ := json.Marshal(nocobaseAuthPayload)

	req, _ := http.NewRequest("POST", nocobaseAuth, strings.NewReader(string(nocobaseAuthPayloadJson)))

	req.Header.Add("Content-Type", "application/json")

	authResponse, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}

	defer authResponse.Body.Close()
	body, err := io.ReadAll(authResponse.Body)
	if err != nil {
		return err
	}

	authResp := new(nocobaseAuthResp)
	err = json.Unmarshal(body, authResp)
	if err != nil {
		return err
	}

	token := "Bearer " + authResp.Data.Token

	for _, r := range userInfos {
		userInfoPayload := &nocobaseUserInfo{
			Username: r.Mobile,
			Email:    r.Email,
			Phone:    r.Mobile,
			Nickname: r.Name,
			Name:     r.Name,
			Userid:   r.Userid,
			Unionid:  r.Unionid,
			Password: r.Mobile,
		}
		nocobaseUserInfoPayloadJson, _ := json.Marshal(userInfoPayload)

		req, _ := http.NewRequest("POST", nocobaseUserCreate, strings.NewReader(string(nocobaseUserInfoPayloadJson)))

		req.Header.Add("Authorization", token)
		req.Header.Add("Content-Type", "application/json")

		_, err := http.DefaultClient.Do(req)
		if err != nil {
			// do nothing
		}
	}

	return nil
}

// @Summary 退出登录
// @Description 获取token
// LoginHandler can be used by clients to get a jwt token.
// Reply will be of the form {"token": "TOKEN"}.
// @Accept  application/json
// @Product application/json
// @Success 200 {string} string "{"code": 200, "msg": "成功退出系统" }"
// @Router /logout [post]
// @Security
func LogOut(c *gin.Context) {
	var loginlog system.LoginLog
	ua := user_agent.New(c.Request.UserAgent())
	loginlog.Ipaddr = c.ClientIP()
	location := tools.GetLocation(c.ClientIP())
	loginlog.LoginLocation = location
	loginlog.LoginTime = tools.GetCurrntTime()
	loginlog.Status = "0"
	loginlog.Remark = c.Request.UserAgent()
	browserName, browserVersion := ua.Browser()
	loginlog.Browser = browserName + " " + browserVersion
	loginlog.Os = ua.OS()
	loginlog.Platform = ua.Platform()
	loginlog.Username = tools.GetUserName(c)
	loginlog.Msg = "退出成功"
	_, _ = loginlog.Create()
	c.JSON(http.StatusOK, gin.H{
		"code": 200,
		"msg":  "退出成功",
	})

}

func Authorizator(data interface{}, c *gin.Context) bool {

	if v, ok := data.(map[string]interface{}); ok {
		u, _ := v["user"].(system.SysUser)
		r, _ := v["role"].(system.SysRole)
		c.Set("role", r.RoleName)
		c.Set("roleIds", r.RoleId)
		c.Set("userId", u.UserId)
		c.Set("userName", u.UserName)

		return true
	}
	return false
}

func Unauthorized(c *gin.Context, code int, message string) {
	c.JSON(http.StatusOK, gin.H{
		"code": code,
		"msg":  message,
	})
}

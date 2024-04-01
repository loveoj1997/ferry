package customers

import (
	"ferry/pkg/customer"
	"ferry/tools"
	"ferry/tools/app"
	"fmt"
	"github.com/gin-gonic/gin"
	uuid "github.com/satori/go.uuid"
	"github.com/spf13/viper"
	"io/ioutil"
	"os"
	"strings"
)

// 客户列表
func CustomersList(c *gin.Context) {
	var (
		err        error
		req        customer.CustomersListRequest
		totalCount int
	)
	err = c.ShouldBind(&req)
	if err != nil {
		app.Error(c, -1, err, "")
		return
	}

	if req.Page == 0 {
		req.Page = 1
	}

	if req.Size == 0 {
		req.Size = 10
	}

	cu := customer.Customer{
		GinObj: c,
	}

	cu.CustomerList(req)

	app.OK(c, map[string]interface{}{
		"data":        taskData,
		"page":        req.Page,
		"per_page":    req.PerPage,
		"total_count": totalCount,
	}, "")
}

// 新建客户
func CreateCustomers(c *gin.Context) {
	var (
		err       error
		taskValue struct {
			Name     string `json:"name"`
			Classify string `json:"classify"`
			Content  string `json:"content"`
		}
	)

	err = c.ShouldBind(&taskValue)
	if err != nil {
		app.Error(c, -1, err, "")
		return
	}

	uuidValue := uuid.Must(uuid.NewV4(), err)
	fileName := fmt.Sprintf("%v/%v-%v-%v",
		viper.GetString("script.path"),
		taskValue.Name,
		strings.Split(uuidValue.String(), "-")[4],
		tools.GetUserName(c),
	)
	if taskValue.Classify == "python" {
		fileName = fileName + ".py"
	} else if taskValue.Classify == "shell" {
		fileName = fileName + ".sh"
	}

	err = ioutil.WriteFile(fileName, []byte(taskValue.Content), 0755)
	if err != nil {
		app.Error(c, -1, err, fmt.Sprintf("创建任务脚本失败: %v", err.Error()))
		return
	}

	app.OK(c, "", "任务创建成功")
}

// 更新客户/联系人
func UpdateCustomers(c *gin.Context) {
	var (
		err  error
		file struct {
			Name     string `json:"name"`
			FullName string `json:"full_name"`
			Classify string `json:"classify"`
			Content  string `json:"content"`
		}
	)

	err = c.ShouldBind(&file)
	if err != nil {
		app.Error(c, -1, err, "")
		return
	}

	fullNameList := strings.Split(file.FullName, "-")
	if fullNameList[0] != file.Name {
		fullNameList[0] = file.Name
	}
	var suffixName string
	if strings.ToLower(file.Classify) == "python" {
		suffixName = ".py"
	} else if strings.ToLower(file.Classify) == "shell" {
		suffixName = ".sh"
	}

	if fullNameList[len(fullNameList)-1][len(fullNameList[len(fullNameList)-1])-3:len(fullNameList[len(fullNameList)-1])] != suffixName {
		tList := strings.Split(fullNameList[len(fullNameList)-1], ".")
		tList[len(tList)-1] = suffixName[1:len(suffixName)]
		fullNameList[len(fullNameList)-1] = strings.Join(tList, ".")
	}

	fileFullName := strings.Join(fullNameList, "-")

	// 修改文件内容
	err = ioutil.WriteFile(fmt.Sprintf("%v/%v", viper.GetString("script.path"), fileFullName), []byte(file.Content), 0666)
	if err != nil {
		app.Error(c, -1, err, fmt.Sprintf("更新脚本文件失败，%v", err.Error()))
		return
	}

	// 修改文件名称
	err = os.Rename(
		fmt.Sprintf("%v/%v", viper.GetString("script.path"), file.FullName),
		fmt.Sprintf("%v/%v", viper.GetString("script.path"), fileFullName),
	)
	if err != nil {
		app.Error(c, -1, err, fmt.Sprintf("更改脚本文件名称失败，%v", err.Error()))
		return
	}

	app.OK(c, "", "更新成功")
}

// 删除客户
func DeleteCustomers(c *gin.Context) {
	fullName := c.DefaultQuery("full_name", "")
	if fullName == "" || strings.Contains(fullName, "/") {
		app.Error(c, -1, errors.New("参数不正确，请确定参数full_name是否传递"), "")
		return
	}

	err := os.RemoveAll(fmt.Sprintf("%v/%v", viper.GetString("script.path"), fullName))
	if err != nil {
		app.Error(c, -1, err, fmt.Sprintf("删除文件失败，%v", err.Error()))
		return
	}

	app.OK(c, nil, "任务删除成功")
}

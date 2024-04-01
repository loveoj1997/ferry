package customer

import (
	"ferry/models/customers"
	"github.com/gin-gonic/gin"
)

type Customer struct {
	GinObj *gin.Context
}

type connectsInfo struct {
	customers.ConnectsInfo
	Tags []string `json:"tags"`
}

type customerInfo struct {
	customers.CustomersInfo
	Contacts []connectsInfo `json:"contacts"`
}

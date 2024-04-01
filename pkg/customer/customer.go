package customer

type CustomersListRequest struct {
	Name        *string `json:"name" form:"name"`
	ContactName *string `json:"contact_name" form:"contact_name"`
	Page        int     `json:"page" form:"page"`
	Size        int     `json:"size" form:"size"`
}

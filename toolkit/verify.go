package toolkit

var (
	IdVerify                  = Rules{"ID": {NotEmpty()}}
	ApiVerify                 = Rules{"Path": {NotEmpty()}, "Description": {NotEmpty()}, "ApiGroup": {NotEmpty()}, "Method": {NotEmpty()}}
	MenuVerify                = Rules{"Path": {NotEmpty()}, "ParentId": {NotEmpty()}, "Name": {NotEmpty()}, "Component": {NotEmpty()}, "Sort": {Ge("0")}}
	MenuMetaVerify            = Rules{"Title": {NotEmpty()}}
	LoginVerify               = Rules{"Username": {NotEmpty()}, "Password": {NotEmpty()}}
	UserCreateVerify          = Rules{"Username": {NotEmpty()}, "NickName": {NotEmpty()}, "Password": {NotEmpty()}}
	PageInfoVerify            = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}}
	PurchasePageInfo          = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}, "PurchaseOrderId": {NotEmpty()}}
	InventorySheetSkuPageInfo = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}, "InventorySheetId": {NotEmpty()}}
	SalesPageInfo             = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}, "SalesOrderId": {NotEmpty()}}
	PurchaseReturnPageInfo    = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}, "PurchaseReturnId": {NotEmpty()}}
	ReturnOrderPageInfo       = Rules{"Page": {NotEmpty()}, "PageSize": {NotEmpty()}, "ReturnOrderId": {NotEmpty()}}
	CustomerVerify            = Rules{"CustomerName": {NotEmpty()}, "CustomerPhoneData": {NotEmpty()}}
	AuthorityVerify           = Rules{"AuthorityId": {NotEmpty()}, "AuthorityName": {NotEmpty()}, "ParentId": {NotEmpty()}}
	AuthorityIdVerify         = Rules{"AuthorityId": {NotEmpty()}}
	OldAuthorityVerify        = Rules{"OldAuthorityId": {NotEmpty()}}
	ChangePasswordVerify      = Rules{"Username": {NotEmpty()}, "Password": {NotEmpty()}, "NewPassword": {NotEmpty()}}
	SetUserAuthorityVerify    = Rules{"AuthorityId": {NotEmpty()}}
)

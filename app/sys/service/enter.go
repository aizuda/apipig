package service

type SysServiceGroup struct {
	WebService
	UserService
	RoleService
	ResourceService
	ResourceApiService
}

var SysService = new(SysServiceGroup)

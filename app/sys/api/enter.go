package api

type SysApiGroup struct {
	WebApi
	UserApi
	RoleApi
	ResourceApi
	ResourceApiApi
}

var SysApi = new(SysApiGroup)

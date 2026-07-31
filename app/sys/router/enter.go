package router

type SysRouterGroup struct {
	WebRouter
	UserRouter
	RoleRouter
	ResourceRouter
	ResourceApiRouter
}

var SysRouter = new(SysRouterGroup)

import { get, post } from './request'

export interface PageResult<T> {
  total: number
  page: number
  pageSize: number
  records: T[]
}

export interface BaseRecord {
  id?: string
  createdAt?: number
  updatedAt?: number
  createdBy?: string
  updatedBy?: string
}

export interface SystemRole extends BaseRecord {
  name: string
  alias: string
  remark?: string
  status: number
  sort: number
}

export interface RoleDetail extends SystemRole {
  resourceIds: string[]
}

export interface SystemUser extends BaseRecord {
  username: string
  realName?: string
  nickName?: string
  avatar?: string
  sex?: number
  phone?: string
  email?: string
  status: number
  jobNum?: string
  loginTime?: number
  pwdTime?: number
}

export interface UserDetail extends SystemUser {
  roleIds: string[]
  departmentIds: string[]
}

export interface ResourceTreeNode extends BaseRecord {
  pid?: string
  title: string
  alias?: string
  type: number
  code?: string
  redirect?: string
  path?: string
  icon?: string
  status: number
  sort?: number
  component?: string
  color?: string
  hidden: boolean
  parentRoute?: string
  keepAlive: boolean
  query?: string
  children?: ResourceTreeNode[]
}

export interface ResourceMenuMeta {
  title?: string
  icon?: string
  type?: number
  hidden?: boolean
  parentRoute?: string
  order?: number
}

export interface ResourceMenuNode {
  name: string
  redirect?: string
  path: string
  component?: string
  meta?: ResourceMenuMeta
  children?: ResourceMenuNode[]
}

export interface ResourceMenuResult {
  menus: ResourceMenuNode[]
  permissions: string[]
}

export interface ResourceSaveParams {
  id?: string
  pid: string
  title: string
  alias?: string
  type: number
  code?: string
  redirect?: string
  path: string
  icon?: string
  status: number
  sort: number
  component?: string
  color?: string
  hidden: boolean
  parentRoute?: string
  keepAlive: boolean
  query?: string
}

export interface UserPageParams {
  page: number
  pageSize: number
  username?: string
  realName?: string
  phone?: string
  status?: number
}

export interface RolePageParams {
  page: number
  pageSize: number
  name?: string
  alias?: string
  status?: number
}

export interface UserSaveParams {
  id?: string
  username: string
  password?: string
  realName?: string
  nickName?: string
  sex?: number
  phone?: string
  email?: string
  jobNum?: string
  roleIds: string[]
}

export interface RoleSaveParams {
  id?: string
  name: string
  alias: string
  remark?: string
  status: number
  sort: number
  resourceIds: string[]
}

export const systemApi = {
  userPage: (params: UserPageParams) => post<PageResult<SystemUser>>('/sys/user/page', params),
  userInfo: () => get<SystemUser & { id: string }>('/sys/user/info'),
  userGet: (id: string) => get<UserDetail>(`/sys/user/get?id=${encodeURIComponent(id)}`),
  userSave: (params: UserSaveParams) => post<boolean>('/sys/user/assign-set', params),
  userUpdate: (params: Partial<SystemUser> & { id: string }) =>
    post<boolean>('/sys/user/update', params),
  userDelete: (ids: string[]) => post<boolean>('/sys/user/delete', { ids }),
  userResetPassword: (ids: string[], password: string) =>
    post<boolean>('/sys/user/reset-password', { ids, password }),
  rolePage: (params: RolePageParams) => post<PageResult<SystemRole>>('/sys/role/page', params),
  roleList: () => post<SystemRole[]>('/sys/role/list', {}),
  roleGet: (id: string) => get<RoleDetail>(`/sys/role/get?id=${encodeURIComponent(id)}`),
  roleSave: (params: RoleSaveParams) => post<boolean>('/sys/role/resource-set', params),
  roleUpdate: (params: Pick<SystemRole, 'id' | 'status'>) =>
    post<boolean>('/sys/role/update', params),
  roleDelete: (ids: string[]) => post<boolean>('/sys/role/delete', { ids }),
  resourceMenu: () => post<ResourceMenuResult>('/sys/resource/list-menu', {}),
  resourceTree: () => post<ResourceTreeNode[]>('/sys/resource/list-tree', {}),
  resourceGet: (id: string) =>
    get<ResourceTreeNode>(`/sys/resource/get?id=${encodeURIComponent(id)}`),
  resourceCreate: (params: ResourceSaveParams) => post<boolean>('/sys/resource/create', params),
  resourceUpdate: (params: Partial<ResourceSaveParams> & { id: string }) =>
    post<boolean>('/sys/resource/update', params),
  resourceDelete: (ids: string[]) => post<boolean>('/sys/resource/delete', { ids }),
}

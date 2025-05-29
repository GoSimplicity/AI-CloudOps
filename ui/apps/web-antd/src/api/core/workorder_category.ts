import { requestClient } from '#/api/request';

// 分类实体
export interface Category {
  id: number;
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  status: number;
  description: string;
  creator_id?: number;
  creator_name?: string;
  created_at?: string;
  updated_at?: string;
  children?: Category[];
  parent?: Category | null;
}

// 分类响应结构
export interface CategoryResp {
  id: number;
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  status: number;
  description: string;
  created_at: string;
  updated_at: string;
  creator_name: string;
  children?: CategoryResp[];
}

// 创建分类请求结构
export interface CreateCategoryReq {
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  description: string;
}

// 更新分类请求结构
export interface UpdateCategoryReq {
  id: number;
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  description: string;
  status: number;
}

// 删除分类请求结构
export interface DeleteCategoryReq {
  id: number;
}

// 列表请求结构
export interface ListCategoryReq {
  page: number;
  size: number;
  status?: number;
}

// 详情请求结构
export interface DetailCategoryReq {
  id: number;
}

// 分类树请求结构
export interface TreeCategoryReq {
  status?: number;
}

// 批量更新状态请求结构
export interface BatchUpdateStatusReq {
  ids: number[];
  status: number;
}

// 分类相关API接口

// 创建分类
export async function createCategory(data: CreateCategoryReq) {
  return requestClient.post('/workorder/category/create', data);
}

// 更新分类
export async function updateCategory(data: UpdateCategoryReq) {
  return requestClient.put(`/workorder/category/update/${data.id}`, data);
}

// 删除分类
export async function deleteCategory(data: DeleteCategoryReq) {
  return requestClient.delete(`/workorder/category/delete/${data.id}`);
}

// 获取分类列表
export async function listCategory(data: ListCategoryReq) {
  return requestClient.get('/workorder/category/list', { params: data });
}

// 获取分类详情
export async function detailCategory(data: DetailCategoryReq) {
  return requestClient.get(`/workorder/category/detail/${data.id}`);
}

// 获取分类树
export async function getCategoryTree(data?: TreeCategoryReq) {
  return requestClient.get('/workorder/category/tree', { params: data });
}
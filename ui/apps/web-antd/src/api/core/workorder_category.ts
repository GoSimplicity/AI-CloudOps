import { requestClient } from '#/api/request';

// 分类实体
export interface Category {
  id: number;
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  status: number | 1 | 2;
  description: string;
  creator_id?: number;
  creator_name?: string;
  created_at?: string;
  updated_at?: string;
}

// 创建分类请求结构
export interface CreateCategoryReq {
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  description: string;
  status?: number | 1 | 2;
}

// 更新分类请求结构
export interface UpdateCategoryReq {
  id: number;
  name: string;
  parent_id?: number | null;
  icon: string;
  sort_order: number;
  description: string;
  status: number | 1 | 2;
}

// 删除分类请求结构
export interface DeleteCategoryReq {
  id: number;
}

// 列表请求结构
export interface ListCategoryReq {
  page: number;
  size: number;
  search?: string;
  status?: number | 1 | 2;
}

// 详情请求结构
export interface DetailCategoryReq {
  id: number;
}

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

// 获取分类统计
export async function getCategoryStatistics() {
  return requestClient.get('/workorder/category/statistics');
}
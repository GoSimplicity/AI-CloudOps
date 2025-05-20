import { requestClient } from '#/api/request';

export interface ClustersItem {
  restricted_name_space: string[];
  kube_config_content: string;
  id: number;
  name: string;
  name_zh: string;
  user_id: number;
  status: string;
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
  version: string;
  env: string;
  api_server_addr: string;
  action_timeout_seconds: number;
  created_at: string;
}

export interface createClusterReq {
  name: string;
  name_zh: string;
  version: string;
  env: string;
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
  restricted_name_space: string[];
  api_server_addr: string;
  kube_config_content: string;
  action_timeout_seconds: number;
}

export interface updateClusterReq {
  id: number;
  name: string;
  name_zh: string;
  version: string;
  env: string;
  cpu_request: string;
  cpu_limit: string;
  memory_request: string;
  memory_limit: string;
  restricted_name_space: string[];
  api_server_addr: string;
  kube_config_content: string;
  action_timeout_seconds: number;
}

export interface NodesItems {
  name: string;
  cluster_id: number;
  status: string;
  roles: string[];
  ip: string;
  pod_num_info: string;
  created_at: string;
}

export interface AddNodeLabelReq {
  cluster_id: number;
  mod_type: string;
  node_name: string[];
  labels: string[];
}

export interface DeleteNodeLabelReq {
  cluster_id: number;
  mod_type: string;
  node_name: string[];
  labels: string[];
}

interface Event {
  type: string;
  component: string;
  reason: string;
  message: string;
  first_time: string;
  last_time: string;
  object: string;
  count: number;
}

export interface GetNodeDetailRes {
  name: string;
  cluster_id: number;
  status: string;
  schedulable: boolean;
  roles: string[];
  age: string;
  ip: string;
  pod_num: number;
  cpu_request_info: string;
  cpu_limit_info: string;
  cpu_usage_info: string;
  memory_request_info: string;
  memory_limit_info: string;
  memory_usage_info: string;
  pod_num_info: string;
  cpu_cores: string;
  mem_gibs: string;
  ephemeral_storage: string;
  kubelet_version: string;
  cri_version: string;
  os_version: string;
  kernel_version: string;
  labels: string[];
  taints: string;
  events: Event[];
}

export interface createNamespaceReq {
  cluster_id: number;
  namespace: string;
  labels: string[];
  annotations: string[];
}

export interface updateNamespaceReq {
  cluster_id: number;
  namespace: string;
  labels: string[];
  annotations: string[];
}

export interface getNamespaceDetailsRes {
  name: string;
  uid: string;
  status: string;
  creation_time: string;
  labels: string[];
  annotations: string[];
}

export async function getAllClustersApi() {
  return requestClient.get('/k8s/clusters/list');
}

export async function getClusterApi(id: number) {
  return requestClient.get(`/k8s/clusters/${id}`);
}

export async function createClusterApi(data: createClusterReq) {
  return requestClient.post('/k8s/clusters/create', data);
}

export async function updateClusterApi(data: updateClusterReq) {
  return requestClient.post('/k8s/clusters/update', data);
}

export async function deleteClusterApi(id: number) {
  return requestClient.delete(`/k8s/clusters/delete/${id}`);
}

export async function batchDeleteClusterApi(data: number[]) {
  return requestClient.delete('/k8s/clusters/batch_delete', { data });
}

export async function getNodeListApi(id: number) {
  return requestClient.get<NodesItems>(`/k8s/nodes/list/${id}`);
}

export async function getNodeDetailsApi(path: string, query: string) {
  return requestClient.get<GetNodeDetailRes>(`/k8s/nodes/${path}?id=${query}`);
}

export async function addNodeLabelApi(data: AddNodeLabelReq) {
  return requestClient.post('/k8s/nodes/labels/add', data);
}

export async function deleteNodeLabelApi(data: DeleteNodeLabelReq) {
  return requestClient.post('/k8s/nodes/labels/add', data);
}

export async function getAllNamespacesApi() {
  return requestClient.get<string[]>(`/k8s/namespaces/list`);
}

export async function getNamespacesByClusterIdApi(id: number) {
  return requestClient.get(`/k8s/namespaces/select/${id}`);
}

export async function createNamespaceApi(data: createClusterReq) {
  return requestClient.post('/k8s/namespaces/create', data);
}

export async function deleteNamespaceApi(id: number, name: string) {
  return requestClient.delete(`/k8s/namespaces/delete/${id}?${name}`);
}

export async function getNamespaceDetails(id: number, name: string) {
  return requestClient.get<getNamespaceDetailsRes>(`/k8s/namespaces/${id}?name=${name}`);
}

export async function updateNamespaceApi(data: updateNamespaceReq) {
  return requestClient.post('/k8s/namespaces/update', data);
}

export async function getPodResources(id: number, name: string) {
  return requestClient.get<string[]>(
    `/k8s/namespaces/${id}/resources?name=${name}`,
  );
}

export async function getNamespaceEvents(id: number, name: string) {
  return requestClient.get(`/k8s/namespaces/${id}/events?name=${name}`);
}

export async function addNodeTaintApi(data: any) {
  return requestClient.post('/k8s/taints/add', data);
}

export async function checkTaintYamlApi(data: any) {
  return requestClient.post('/k8s/taints/taint_check', data);
}

export async function setNodeScheduleApi(data: any) {
  return requestClient.post('/k8s/taints/enable_switch', data);
}

export async function deleteNodeTaintApi(data: any) {
  return requestClient.delete('/k8s/taints/delete', data);
}

export async function clearNodeTaintsApi(data: any) {
  return requestClient.post('/k8s/taints/drain', data);
}

export async function getPodsByNamespaceApi(id: number, namespace: string) {
  return requestClient.get(`/k8s/pods/${id}?namespace=${namespace}`);
}

export async function getContainersByPodNameApi(id: number, podName: string, namespace: string) {
  return requestClient.get(`/k8s/pods/${id}/${podName}/containers?namespace=${namespace}`);
}

export async function getContainerLogsApi(id: number, podName: string, container: string, namespace: string) {
  return requestClient.get(`/k8s/pods/${id}/${podName}/${container}/logs?namespace=${namespace}`);
}

export async function getPodYamlApi(id: number, podName: string, namespace: string) {
  return requestClient.get(`/k8s/pods//${id}/${podName}/yaml?namespace=${namespace}`);
}

export async function deletePodApi(id: number, podName: string, namespace: string) {
  return requestClient.delete(`/k8s/pods/delete/${id}?podName=${podName}&namespace=${namespace}`);
}

export async function getServiceListApi(id: number, namespace: string) {
  return requestClient.get(`/k8s/services/${id}?namespace=${namespace}`);
}

export async function getServiceYamlApi(id: number, svcName: string, namespace: string) {
  return requestClient.get(`/k8s/services/${id}/${svcName}/yaml?namespace=${namespace}`);
}

export async function updateServiceApi(data: any) {
  return requestClient.post('/k8s/services/update', data);
}

export async function deleteServiceApi(id: number, namespace: string, svcName: string) {
  return requestClient.delete(`/k8s/services/delete/${id}?namespace=${namespace}&svcName=${svcName}`);
}

export async function getDeployListApi(id: number, namespace: string) {
  return requestClient.get(`/k8s/deployments/${id}?namespace=${namespace}`);
}

export async function getDeployYamlApi(id: number, deployment_name: string, namespace: string) {
  return requestClient.get(`/k8s/deployments/${id}/yaml?namespace=${namespace}&deployment_name=${deployment_name}`);
}

export async function deleteDeployApi(id: number, namespace: string, deployment_name: string) {
  return requestClient.delete(`/k8s/deployments/delete/${id}?namespace=${namespace}&deployment_name=${deployment_name}`);
}

export async function restartDeployApi(id: number, namespace: string, deployment_name: string) {
  return requestClient.post(`/k8s/deployments/restart/${id}?namespace=${namespace}&deployment_name=${deployment_name}`);
}

export async function getConfigMapListApi(id: number, namespace: string) {
  return requestClient.get(`/k8s/configmaps/${id}?namespace=${namespace}`);
}

export async function getConfigMapYamlApi(id: number, configmap_name: string, namespace: string) {
  return requestClient.get(`/k8s/configmaps/${id}/yaml?namespace=${namespace}&configmap_name=${configmap_name}`);
}

export async function deleteConfigMapApi(id: number, namespace: string, configmap_name: string) {
  return requestClient.delete(`/k8s/configmaps/delete/${id}?namespace=${namespace}&configmap_name=${configmap_name}`);
}

export async function getYamlTemplateApi(cluster_id: number) {
  return requestClient.get(`/k8s/yaml_templates/list/?cluster_id=${cluster_id}`);
}

export async function createYamlTemplateApi(data: any) {
  return requestClient.post('/k8s/yaml_templates/create', data);
}

export async function updateYamlTemplateApi(data: any) {
  return requestClient.post('/k8s/yaml_templates/update', data);
}

export async function deleteYamlTemplateApi(id: number, cluster_id: number) {
  return requestClient.delete(`/k8s/yaml_templates/delete/${id}?cluster_id=${cluster_id}`);
}

export async function getYamlTemplateDetailApi(id: number, cluster_id: number) {
  return requestClient.get(`/k8s/yaml_templates/${id}/yaml?cluster_id=${cluster_id}`);
}

export async function checkYamlTemplateApi(data: any) {
  return requestClient.post('/k8s/yaml_templates/check', data);
}

export async function getYamlTaskListApi() {
  return requestClient.get('/k8s/yaml_tasks/list');
}

export async function deleteYamlTaskApi(id: number) {
  return requestClient.delete(`/k8s/yaml_tasks/delete/${id}`);
}

export async function createYamlTaskApi(data: any) {
  return requestClient.post('/k8s/yaml_tasks/create', data);
}

export async function updateYamlTaskApi(data: any) {
  return requestClient.post('/k8s/yaml_tasks/update', data);
}

export async function applyYamlTaskApi(id: number) {
  return requestClient.post(`/k8s/yaml_tasks/apply/${id}`);
}

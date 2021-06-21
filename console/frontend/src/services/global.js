import request from '@/utils/request';

export async function queryCurrentUser() {
  return request('/api/v1/current-user');
}
export async function queryConfig() {
  return request('/api/v1/dlc/common-config');
}
export async function queryNamespaces() {
  return request('/api/v1/dlc/namespaces');
}

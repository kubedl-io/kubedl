import request from '@/utils/request';
import Qs from 'qs';

const APIV1Prefix = '/api/v1';

// new datasource config
export async function newWorkspace(params) {
  return request(`${APIV1Prefix}/workspace/create`, {
    method: 'POST',
    body: Qs.stringify(params),
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8',
    },
  });
}

import request from '@/utils/request';
import Qs from 'qs';

const APIV1Prefix = '/api/v1';

// new code config
export async function newGitSource(params) {
  return request(`${APIV1Prefix}/codesource`, {
    method: 'POST',
    body: Qs.stringify(params),
    headers: {
      'Content-Type': 'application/x-www-form-urlencoded;charset=UTF-8',
    },
  });
}

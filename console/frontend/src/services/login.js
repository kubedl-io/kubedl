import request from '@/utils/request';

export async function fakeAccountLogin(params) {
  return request('/api/login/account', {
    method: 'POST',
    data: params,
  });
}
export async function getFakeCaptcha(mobile) {
  return request(`/api/login/captcha?mobile=${mobile}`);
}

export async function queryLogin(params) {
  return request('/api/v1/login/oauth2', {
    method: 'POST',
    data: params,
  });
}

export async function queryLoginOut() {
  return request('/api/v1/logout');
}

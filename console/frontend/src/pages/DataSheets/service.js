import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

// get datasource config
export async function getDatasources() {
  return request(`${APIV1Prefix}/datasource`);
}

// // get datasource config by name
// export async function getDatasourceByName(name) {
//   return request(`${APIV1Prefix}/datasource/${name}`);
// }

// get codesource config
export async function getCodesources() {
  return request(`${APIV1Prefix}/codesource`);
}
export async function deleteDataConfig({ name }) {
  return request(
    `${APIV1Prefix}/datasource/${name}`,
    {
      method: 'DELETE',
    },
  );
}
export async function deleteGitConfig({ name }) {
  return request(
    `${APIV1Prefix}/codesource/${name}`,
    {
      method: 'DELETE',
    },
  );
}

import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

export async function submitJob(data, kind) {
  return request(`${APIV1Prefix}/job/submit`, {
    method: 'POST',
    params: {
      kind,
    },
    data,
  });
}

export async function listPVC(namespace) {
  return request(`${APIV1Prefix}/pvc/list`, {
    params: {
      namespace,
    },
  });
}

export async function getDatasources() {
  return request(`${APIV1Prefix}/datasource`);
}

export async function getCodeSource() {
  return request(`${APIV1Prefix}/codesource`);
}

export async function getNamespaces() {
  return request(`${APIV1Prefix}/kubedl/namespaces`);
}

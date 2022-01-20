import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

export async function submitNotebook(data, kind) {
  return request(`${APIV1Prefix}/notebook/submit`, {
    method: 'POST',
    params: {
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

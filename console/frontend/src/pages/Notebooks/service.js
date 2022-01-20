import request from "@/utils/request";

const APIV1Prefix = "/api/v1";

export async function queryNotebooks(params) {
  const ret = await request(`${APIV1Prefix}/notebook/list`, {
    params,
  });
  return {
    data: ret.data.notebookInfos,
    total: ret.data.total,
  };
}

export async function deleteNotebook(namespace, name, id) {
  return request(
    `${APIV1Prefix}/notebook/${namespace}/${name}?id=${id}`,
    {
      method: "DELETE",
    }
  );
}

export async function stopNotebook(deployRegion, namespace, name) {
  return request(
    `${APIV1Prefix}/notebook/stop/${deployRegion}/${namespace}/${name}`,
    {
      method: "POST",
    }
  );
}

export async function submitNotebook(data) {
  return request(`${APIV1Prefix}/notebook/submit`, {
    method: "POST",
    body: data,
  });
}

export async function cloneInfoNotebook(nameSpace, notebookName) {
  return request(
      `${APIV1Prefix}/notebook/json/${nameSpace}/${notebookName}`,
  );
}

import request from "@/utils/request";

const APIV1Prefix = "/api/v1";

export async function queryWorkspaces(params) {
  const ret = await request(`${APIV1Prefix}/workspace/list`, {
    params,
  });
  return {
    data: ret.data.workspaceInfos,
    total: ret.data.total,
  };
}

export async function deleteWorkspace(name) {
  return request(
    `${APIV1Prefix}/workspace/${name}`,
    {
      method: "DELETE",
    }
  );
}

export async function createWorkspace(data) {
  return request(`${APIV1Prefix}/notebook/create`, {
    method: "POST",
    body: data,
  });
}

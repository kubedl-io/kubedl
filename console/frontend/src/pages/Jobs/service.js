import request from "@/utils/request";

const APIV1Prefix = "/api/v1";

export async function queryJobs(params) {
  const ret = await request(`${APIV1Prefix}/job/list`, {
    params,
  });
  return {
    data: ret.data.jobInfos,
    total: ret.data.total,
  };
}

export async function deleteJobs(namespace, name, id, kind, submitTime) {
  return request(
    `${APIV1Prefix}/job/${namespace}/${name}?kind=${kind}&id=${id}`,
    {
      method: "DELETE",
    }
  );
}

export async function stopJobs(deployRegion, namespace, name) {
  return request(
    `${APIV1Prefix}/job/stop/${deployRegion}/${namespace}/${name}`,
    {
      method: "POST",
    }
  );
}

export async function submitJob(data) {
  return request(`${APIV1Prefix}/job/submit`, {
    method: "POST",
    body: data,
  });
}

export async function getJobTensorboardStatus(params) {
  return request(`${APIV1Prefix}/tensorboard/status`, {
    params,
  });
}
export async function reApplyJobTensorboard(params, data) {
  return request(`${APIV1Prefix}/tensorboard/reapply`, {
    method: "POST",
    params,
    data,
  });
}

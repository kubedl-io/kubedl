import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

export async function getJobDetail(params) {
  return request(`${APIV1Prefix}/job/detail`, {
    params: {
      ...params,
      page_size: 10,
      replica_type: 'ALL',
      status: 'ALL',
    },
  });
}

export async function getPodLog(namespace, podName, podId, podCreateTime) {
  return request(`${APIV1Prefix}/log/logs/${namespace}/${podName}`, {
    params: {
      fromTime: podCreateTime,
      uid: podId,
    },
  });
}

export function downloadLogURL(namespace, podName, podId, podCreateTime) {
  const fromTime = podCreateTime;
  return encodeURI(
    `${APIV1Prefix
    }/log/download/${namespace}/${podName}?fromTime=${fromTime}&uid=${podId}`,
  );
}

export async function getEvents(namespace, name, id, podCreateTime) {
  return request(`${APIV1Prefix}/event/events/${namespace}/${name}`, {
    params: {
      fromTime: podCreateTime,
      uid: id,
    },
  });
}

export async function deleteJobs(namespace, name, id, kind, submitTime) {
  return request(
    `${APIV1Prefix}/job/${namespace}/${name}?kind=${kind}&id=${id}`,
    {
      method: 'DELETE',
    },
  );
}

export async function cloneInfoJobs(nameSpace, jobName, JobType) {
  return request(
    `${APIV1Prefix}/job/json/${nameSpace}/${jobName}?kind=${JobType}`,
  );
}

export async function getPodRangeCpuInfoJobs(name, start, end, step, namespace) {
  return request(
    `${APIV1Prefix}/data/podRangeInfo?query=(1-avg(irate(node_cpu_seconds_total{job="${name}",mode="idle",namespace="${namespace}"}[5m]))by(pod_name))*100&start=${start}&end=${end}&step=${step}`,
  );
}

export async function getPodRangeMemoryInfoJobs(name, start, end, step, namespace) {
  return request(
    `${APIV1Prefix}/data/podRangeInfo?query=(1-(node_memory_MemAvailable_bytes{job="${name}",namespace="${namespace}"}/(node_memory_MemTotal_bytes{job="${name}",namespace="${namespace}"})))*100&start=${start}&end=${end}&step=${step}`,
  );
}

export async function getPodRangeGpuInfoJobs(name, start, end, step, namespace) {
  return request(
    `${APIV1Prefix}/data/podRangeInfo?query=avg(nvidia_gpu_duty_cycle{job="${name}",namespace="${namespace}")by(pod_name)&start=${start}&end=${end}&step=${step}`,
  );
}
// job stop api
export async function stopJobs(nameSpace, jobName, id, JobType) {
  return request(
    `${APIV1Prefix}/job/${nameSpace}/${jobName}?kind=${JobType}&id=${id}`,
  );
}

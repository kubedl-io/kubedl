import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

export async function getOverviewTotal() {
  return request(`${APIV1Prefix}/data/total`);
}

export async function getOverviewRequestPodPhase() {
  return request(`${APIV1Prefix}/data/request/Running`);
}

export async function getOverviewNodeInfos(params) {
  const ret = await request(`${APIV1Prefix}/data/nodeInfos`, {
    params,
  });
  return ret;
}

export async function getTimeStatistics(startTime, endTime) {
  return request(`${APIV1Prefix}/job/statistics?start_time=${startTime}&end_time=${endTime}`);
}

// top task
export async function getTopResourcesStatistics(startTime, limit) {
  return request(`${APIV1Prefix}/job/running-jobs?start_time=${startTime}&limit=${limit}`);
}

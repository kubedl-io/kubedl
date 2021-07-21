import request from '@/utils/request';

const APIV1Prefix = '/api/v1';

export async function queryJobs(params) {
  const ret = await request(`${APIV1Prefix}/job/list`, {
    params,
  });
  return {
    data: ret.data.jobInfos,
    total: ret.data.total,
  };
}

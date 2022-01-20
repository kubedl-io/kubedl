import { Badge } from 'antd';
import React from 'react';

const STATUS_MAP = {
    'All': {
      // text: '全部',
      text: 'All',
      status: 'default',
    },
    'Completed': {
      // text: '已完成',
      text: 'Created',
      status: 'default',
    },
    'Pending': {
      // text: '创建中',
      text: 'Waiting',
      status: 'processing',
    },
    'Running': {
      // text: '运行中',
      text: 'Running',
      status: 'success',
    },
    'Failed': {
      // text: '执行失败',
      text: 'Failed',
      status: 'error',
    },
    'Stopped': {
      // text: '已停止',
      text: 'Stopped',
      status: 'error',
    },
}

const PodStatus = props => {
    const { status } = props
    const s = STATUS_MAP[status] || {
      text: '未知',
      // text: 'Stopped',
      status: 'default',
    }

    return (
        <Badge status={s.status} text={s.text} />
    )
    
};

export default PodStatus;

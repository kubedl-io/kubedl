import { Badge } from 'antd';
import React from 'react';
import { FormattedMessage } from 'umi';

const STATUS_MAP = {
    'All': {
      text: <FormattedMessage id="component.tagSelect.all" />,
      // text: 'All',
      status: 'default',
    },
    'Created': {
      text: <FormattedMessage id="kubedl-dashboard-has-created" />,
      // text: 'Created',
      status: 'default',
    },
    'Waiting': {
      text: <FormattedMessage id="kubedl-dashboard-waiting-for" />,
      // text: 'Waiting',
      status: 'processing',
    },
    'Running': {
      text: <FormattedMessage id="kubedl-dashboard-executing" />,
      // text: 'Running',
      status: 'processing',
    },
    'Succeeded': {
      text: <FormattedMessage id="kubedl-dashboard-execute-success" />,
      // text: 'Succeeded',
      status: 'success',
    },
    'Failed': {
      text: <FormattedMessage id="kubedl-dashboard-execute-failure" />,
      // text: 'Failed',
      status: 'error',
    },
    'Stopped': {
      text: <FormattedMessage id="kubedl-dashboard-has-stopped" />,
      // text: 'Stopped',
      status: 'error',
    },
}

const JobStatus = props => {
    const { status } = props

    const s = STATUS_MAP[status] || {
      text: <FormattedMessage id="kubedl-dashboard-status-unknown" />,
      // text: 'Stopped',
      status: 'default',
    }

    return (
        <Badge status={s.status} text={s.text} />
    )
    
};

export default JobStatus;

import React from "react";
import {
  SyncOutlined
} from "@ant-design/icons";
import { Modal, Spin, Tabs, Empty, Button, Tooltip, Checkbox } from "antd";
import { LazyLog } from "react-lazylog";
import { downloadLogURL, getEvents, getPodLog } from "./service";
import styles from './LogModal.less';
import moment from "moment";
import { FormattedMessage } from 'umi';
class LogModal extends React.Component {
  refreshInterval = null
  state = {
    currentKey: "logs",
    loading: false,
    logContent: '',
    enableRefresh: true,
  }

  componentDidMount() {
    this.onTabChange(this.state.currentKey)
    this.enableRefresh(this.state.enableRefresh)
  }

  enableRefresh (enabled) {
    this.setState({
      enableRefresh: enabled
    })
    if (enabled) {
      const interval = 5 * 1000
      this.refreshInterval = setInterval(() => {
          this.loadLogSilently()
      }, interval)
    } else {
      clearInterval(this.refreshInterval)
    }
  }

  async loadLog() {
    this.setState({
      loading: true,
    })
    await this.loadLogSilently()
    this.setState({
      loading: false
    });
  }

  async loadLogSilently() {
    const { job, pod } = this.props;
    let res;
    let createTime = moment(pod.createTime).toDate().toISOString();
    if (this.state.currentKey === "events") {
      res = await getEvents(job.namespace, pod.name, pod.podId, createTime);
    } else {
      res = await getPodLog(job.namespace, pod.name, pod.podId, createTime);
    }

    if (res.code === "200") {
      let logs = res.data || [];
      this.setState({
        logContent: logs.join('\n')
      });
    }
  }

  componentWillUnmount() {
      clearInterval(this.refreshInterval)
  }

  onTabChange = key => {
    this.setState({
      currentKey: key,
    }, () => {
      this.loadLog()
    });
  };

  handleDownload = e => {
    const { job, pod } = this.props;
    let createTime = moment(pod.createTime).toDate().toISOString();
    window.open(
      downloadLogURL(job.namespace, pod.name, pod.podId, createTime),
      "_blank"
    );
  };

  handleCancel = e => {
    const { onCancel } = this.props;
    onCancel();
  };

  render() {
    return (
      <Modal
        width={1024}
        visible
        footer={[
          <span>
            <Checkbox checked={this.state.enableRefresh} onChange={(e) => this.enableRefresh(e.target.checked)}><FormattedMessage id="kubedl-dashboard-auto-refresh" /></Checkbox>
          </span>,
          <Button key="back" onClick={this.handleCancel}>
            <FormattedMessage id="kubedl-dashboard-close" />
          </Button>,
          <Button key="submit" type="primary" onClick={this.handleDownload}>
            <FormattedMessage id="kubedl-dashboard-download-logs" />
          </Button>,
        ]}
        onCancel={this.handleCancel}
        maskClosable={false}
        keyboard={false}
        width='80vw'
        style={{top: '2vh'}}
      >
        <Tabs
          defaultActiveKey={this.state.currentKey}
          onChange={e => this.onTabChange(e)}
        >
          <Tabs.TabPane tab={<FormattedMessage id="kubedl-dashboard-user-logs" />} key="logs" />
        <Tabs.TabPane tab={<FormattedMessage id="kubedl-dashboard-system-logs" />} key="events" />
        </Tabs>
        <Spin spinning={this.state.loading}>
          {this.state.logContent.length === 0 
          ? <Empty description={<FormattedMessage id="kubedl-dashboard-no-logs" />} />
          : <div className={styles.logContainer} >
              <LazyLog
                selectableLines={true}
                extraLines={1}
                enableSearch
                text={this.state.logContent}
                caseInsensitive
                follow={true}
                onRowClick={(text)=>{
                  let regExp = /(\/\w+)+\.[a-z]+/gi;
                  if(regExp.test(text)){
                      let match = text.match(regExp)[0];
                      window.open(match, "_blank");
                  }
                }}
              />
              <div class={styles.refresh}>
                <Tooltip title={<FormattedMessage id="kubedl-dashboard-refresh" />}>
                  <Button shape="circle" size="large" icon={<SyncOutlined />} onClick={() => this.loadLog()} />
                </Tooltip>
              </div>
            </div>
          }
        </Spin>
      </Modal>
    );
  }
}

export default LogModal;

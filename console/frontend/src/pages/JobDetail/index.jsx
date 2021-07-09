import {Button, Card, Descriptions, Modal, Steps, Table, Empty,Row,Col, Tabs, message} from "antd";
import {PageHeaderWrapper} from "@ant-design/pro-layout";
import {ExclamationCircleOutlined} from "@ant-design/icons";
import React, {Component, Fragment} from "react";
import PageLoading from "@/components/PageLoading";
import JobStatus from "@/components/JobStatus";
import LogModal from "./LogModal";
import { getJobDetail, getEvents, deleteJobs, cloneInfoJobs, getPodRangeCpuInfoJobs, getPodRangeGpuInfoJobs, getPodRangeMemoryInfoJobs, stopJobs } from "./service";
import styles from "./style.less";
import moment from "moment";
import { LazyLog } from "react-lazylog";
import { FormattedMessage,history, useIntl, formatMessage } from 'umi';
import PodCharts from '@/pages/JobDetail/PodCharts';
import {queryCurrentUser} from "@/services/global";
const jobDeleteTitleFormatedText = formatMessage({
    id: 'kubedl-dashboard-delete-job'
});
const jobStopTitleFormatedText = formatMessage({
    id: 'kubedl-dashboard-stop-job'
});
const jobDeleteContentFormatedText = formatMessage({
    id: 'kubedl-dashboard-delete-job-confirm'
});
const jobStopContentFormatedText = formatMessage({
    id: 'kubedl-dashboard-stop-job-confirm'
});
const jobModelOkText = formatMessage({
    id: 'kubedl-dashboard-ok'
});
const jobModelCancelText = formatMessage({
    id: 'kubedl-dashboard-cancel'
});

class JobDetail extends Component {
  refreshInterval = null;
  state = {
    detailLoading: true,
    detail: {},
    eventsLoading: true,
    events: [],
    total: 0,
    tabActiveKey: "spec",
    logModalVisible: false,
    currentPod: undefined,
    currentPage: 1,
    currentPageSize: 10,
    resourceConfigKey: 'Worker',
    podChartsValue: [],
    podChartsType: 'CPU',
    podChartsLoading: false,
    users:{},
  };
  onResourceConfigTabChange = (key, type) => {
    this.setState({ [type]: key });
  };
  async componentDidMount() {
    await this.fetchDetail();
    await this.fetchUser();
    // await this.fetchGetPodRangeInfoJobs(this.state.podChartsType);
    const interval = 5 * 1000;
    this.refreshInterval = setInterval(() => {
        this.fetchDetailSilently()
    }, interval);
  }
  componentWillUnmount() {
    clearInterval(this.refreshInterval);
  }

  async fetchDetail() {
    this.setState({
        detailLoading: true
    });
    await this.fetchDetailSilently();
    this.setState({
        detailLoading: false,
    })
  }

  fetchUser = async () => {
      const users = await queryCurrentUser();
      const userInfos = users.data ? users.data : {};
      this.setState({
          users: userInfos
      });
  }

  async fetchGetPodRangeInfoJobs(type) {
    this.setState({
        podChartsLoading: true
    });
    const startTime = this.state.detail?.createTime && this.state.detail?.createTime !== '' ? new Date(this.state.detail?.createTime) : null;
    const endTime = this.state.detail?.endTime && this.state.detail?.endTime !== '' ? new Date(this.state.detail.endTime) : new Date(new Date().getTime());
    let info = {};
    if (type === 'CPU') {
        info = await getPodRangeCpuInfoJobs(this.state.detail?.name, Math.floor(startTime.valueOf()/ 1000), Math.floor(endTime.valueOf()/ 1000), '10m', sessionStorage.getItem("namespace"))
    }else if (type === 'GPU') {
        info = await getPodRangeGpuInfoJobs(this.state.detail?.name, Math.floor(startTime.valueOf()/ 1000), Math.floor(endTime.valueOf()/ 1000), '10m', sessionStorage.getItem("namespace"));
    }else {
        info = await getPodRangeMemoryInfoJobs(this.state.detail?.name, Math.floor(startTime.valueOf()/ 1000), Math.floor(endTime.valueOf()/ 1000), '10m', sessionStorage.getItem("namespace"));
    }
    const infoPod = info && info.data ? info.data : [];
    this.setState({
        podChartsValue: this.handlePodChartsData(infoPod)
    }, () => {
        this.setState({
            podChartsLoading: false
        });
    })
  }

  handlePodChartsData = data => {
      const newData = [];
      data && data.length > 0 && data.map((p) => {
          p['values'].map((v) => {
              newData.push({
                  name: p['metric']['pod_name'],
                  x: v[0],
                  y: Number(v[1]).toFixed(2),
              })
          })
      });
      return newData;
  }

  async fetchDetailSilently() {
    const { match, location } = this.props;
    try{
        let res = await getJobDetail({
            job_id: match.params.id,
            ...location.query,
            current_page: this.state.currentPage,
            page_size: this.state.currentPageSize
        });
        if(res.code ==200){
            this.setState({
                detail: res.data ? res.data.jobInfo : {},
                total: res.data ? res.data.total : 0,
            }, () => {
                const newResources = this.state.detail && this.state.detail.resources ? eval('('+ this.state.detail?.resources +')') : {};
                this.setState({
                    resourceConfigKey: JSON.stringify(newResources) !== '{}' ? Object.keys(newResources)[0] : '',
                });
            });
        }else {
            message.error(JSON.stringify(res.data));
        }
    }catch(error){
        message.error(JSON.stringify(error));
    }
  }
    async fetchEvents() {
        const { match, location } = this.props;
        const { detail } = this.state;

        this.setState({
            eventsLoading: true
        })

        let createTime = moment(detail.createTime).toDate().toISOString();
        let res = await getEvents(detail.namespace, detail.name, detail.id, createTime)
        this.setState({
            eventsLoading: false,
            events: res.data || []
        })
    }

    onTabChange = tabActiveKey => {
        const {} = this.props;
        const { detail } = this.state;
        this.setState({
            tabActiveKey
        });
        if (tabActiveKey === "events") {
            this.fetchEvents()
        }
    };

    onLog = p => {
        this.setState({
            currentPod: p,
            logModalVisible: true
        });
    };

    onLogClose = () => {
        this.setState({
            currentPod: undefined,
            logModalVisible: false
        });
    };

    onPaginationChange = (page, pageSize) => {
      this.setState({
        currentPage: page,
        currentPageSize: pageSize
      }, () => {
        this.fetchDetail()
      })
    }

    action = detail => {
        let isDisabled;
        if (this.state.users.loginId && this.state.users.loginId !== "") {
            isDisabled = true;
        }
        return (
            <Fragment>
                <Button type="primary" onClick={() => this.fetchJobCloneInfoSilently(detail)} disabled={!isDisabled} >
                    {<FormattedMessage id="kubedl-dashboard-clone" />}
                </Button>
                {/*{*/}
                {/*   detail.jobStatus === 'Running' &&*/}
                {/*    <Button type="primary"*/}
                {/*         style={{background: isDisabled ? '#f2b924' : '', borderColor: isDisabled ? '#f2b924' : ''}}*/}
                {/*          onClick={() => this.onJobStop(detail)}*/}
                {/*           disabled={!isDisabled}>*/}
                {/*       {<FormattedMessage id="kubedl-dashboard-stop" />}*/}
                {/*   </Button>*/}
                {/*}*/}
                <Button type="danger" onClick={() => this.onJobDelete(detail)} disabled={!isDisabled}>
                    {<FormattedMessage id="component.delete" />}
                </Button>
            </Fragment>
        );
    };
    //克隆
    async fetchJobCloneInfoSilently(job) {
        const { match, location } = this.props;
        const namespace = location.query.namespace;
       let res = await cloneInfoJobs(namespace, job.name, job.jobType);
        const infoData = res?.data ?? {};
        try{
            if (JSON.parse(infoData || "{}").metadata) {
                sessionStorage.setItem("job", infoData);
                history.push({
                    pathname: '/job-submit',
                });
            }
        }catch (e) {
            console.log(e)
        }
    }

    onJobStop = job => {
        Modal.confirm({
            title: jobStopTitleFormatedText,
            icon: <ExclamationCircleOutlined/>,
            content:  `${jobStopContentFormatedText} ${job.name} ?`,
            okText: jobModelOkText,
            cancelText: jobModelCancelText,
            onOk: () =>
                stopJobs(
                    job.namespace,
                    job.name,
                    job.id,
                    job.jobType
                ).then(() => {
                    this.fetchDetail();
                    this.fetchGetPodRangeInfoJobs(this.state.podChartsType)
                }),
            onCancel() {
            }
        });
    }

    onJobDelete = job => {
        Modal.confirm({
            title: jobDeleteTitleFormatedText,
            icon: <ExclamationCircleOutlined/>,
            content:  `${jobStopContentFormatedText} ${job.name} ?`,
            okText: jobModelOkText,
            cancelText: jobModelCancelText,
            onOk: () =>
                deleteJobs(
                    job.namespace,
                    job.name,
                    job.id,
                    job.jobType,
                    moment(job.submitTime)
                        .utc()
                        .format()
                ).then(() => {
                    history.replace('/jobs')
                }),
            onCancel() {
            }
        });
    };

    goToDatasheets = () => {
        return history.push({
            pathname: '/datasheets',
        });
    }

    description = detail => {
        const jobConfig = eval('('+ detail.jobConfig +')');
        const jobResources = eval('('+ detail.resources +')');
        let descriptions = (
            <div>
                <Descriptions bordered className={styles.headerList} size="small">
                    <Descriptions.Item label="ID">{detail.id}</Descriptions.Item>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-job-type" />} span={2}>
                        {detail.jobType}
                    </Descriptions.Item>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-creation-time" />}>
                        {detail.createTime}
                    </Descriptions.Item>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-end-time" />}>{detail.endTime}</Descriptions.Item>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-execution-time" />}>
                        {detail.durationTime}
                    </Descriptions.Item>
                </Descriptions>
              {/*
                <Descriptions bordered title={<FormattedMessage id="kubedl-dashboard-job-config" />}>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-data-config" />}>
                         {jobConfig&&jobConfig.data_bindings? jobConfig.data_bindings.map((data,index)=>{
                             let len = jobConfig.data_bindings.length;
                             let regExp = /^data-/g;
                              if(regExp.test(data)){
                                   return <span key ={index} className={styles.detailLink} onClick={this.goToDatasheets}>{`${data.replace(/^data-/g,"")} ${len-1!==index?'  |  ':''}`}</span>
                              };
                              return `${data}${len-1!==index?'  |  ':''}`;
                            }):''
                        }
                    </Descriptions.Item>
                  { environment  &&
                   <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-code-config" />} span={3}>
                        { jobConfig.code_bindings && jobConfig.code_bindings.aliasName ? <div>
                            <span className={styles.detailLink} onClick={this.goToDatasheets}>{jobConfig.code_bindings.aliasName.slice(5)}</span>
                            </div> :
                            <Row>
                                <Col span={12}><FormattedMessage id="kubedl-dashboard-repository-address" />：{jobConfig?.code_bindings?.source ?? ''}</Col>
                                <Col span={12}><FormattedMessage id="kubedl-dashboard-code-branch" />：{jobConfig?.code_bindings?.branch ?? ''}</Col>
                            </Row>
                         }
                    </Descriptions.Item>
                   }
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-execute-command" />} span={3}>
                        {jobConfig?.commands.toString()}
                    </Descriptions.Item>
                    <Descriptions.Item label={<FormattedMessage id="kubedl-dashboard-resource-info" />} span={3}>
                        {
                            this.state.resourceConfigKey !== '' &&  <Card
                                style={{ width: '100%'}}
                                tabList={jobResources && Object.keys(jobResources).map((type)=>{
                                    return {
                                        key:type,
                                        tab:type,
                                    }
                                })}
                                activeTabKey={this.state.resourceConfigKey}
                                onTabChange={key => {
                                    this.onResourceConfigTabChange(key, 'resourceConfigKey');
                                }}
                            >
                                {
                                    jobResources[this.state.resourceConfigKey].resources.limits && Object.keys(jobResources[this.state.resourceConfigKey].resources.limits)
                                        .map((item)=>{
                                            return (
                                                <div>
                                                    <span style={{fontWeight:500}}>{item.indexOf('gpu')!==-1?'gpu':item}:</span>
                                                    &nbsp;&nbsp; &nbsp;&nbsp;
                                                    <span>
                                                    {item.indexOf('memory')!==-1?jobResources[this.state.resourceConfigKey].resources.limits[item].split('Gi')[0] + 'GB': jobResources[this.state.resourceConfigKey].resources.limits[item]}
                                                    </span>
                                                </div>
                                            );
                                        })
                                }
                                {
                                    jobResources[this.state.resourceConfigKey] && jobResources[this.state.resourceConfigKey].replicas !== 0 &&
                                    <span style={{fontWeight:500}}>instance_num：{jobResources[this.state.resourceConfigKey].replicas}</span>
                                }
                            </Card>
                        }
                    </Descriptions.Item>
                </Descriptions>
                    */ }
            </div>
        );
        return (descriptions);
    };
    onPodTabChange = (value) => {
        this.setState({
            podChartsType: value,
            podChartsValue: []
        }, () => {this.fetchGetPodRangeInfoJobs(value);})
    }

    render() {
        const {tabActiveKey, detail, detailLoading, events, eventsLoading, total} = this.state;
        if (detailLoading !== false) {
            return <PageLoading/>;
        }
        const title = (
        <span>
          <span style={{paddingRight: 12}}>
            {detail.namespace} / {detail.name}
          </span>
          <JobStatus status={detail.jobStatus}/>
        </span>
        );

        return (
            <PageHeaderWrapper
                onBack={() => history.goBack()}
                title={title}
                extra={environment && this.action(detail)}
                className={styles.pageHeader}
                content={ detail && this.description(detail)}
                tabActiveKey={tabActiveKey}
                onTabChange={this.onTabChange}
                tabList={[
                    {
                        key: "spec",
                        tab: <FormattedMessage id="kubedl-dashboard-instances" />
                    },
                    {
                        key: "events",
                        tab: <FormattedMessage id="kubedl-dashboard-events" />
                    }
                ]}
            >
                <div className={styles.main}>
                    {this.state.tabActiveKey === "spec" && (
                        <Card bordered={false}>
                            <Table
                                size="small"
                                pagination={{
                                  total: total,
                                  current: this.state.currentPage,
                                  pageSize: this.state.currentPageSize,
                                  onChange: this.onPaginationChange
                                }}
                                columns={[
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-type" />,
                                        dataIndex: "replicaType",
                                        key: "replicaType"
                                    },
                                    {
                                        title: "IP",
                                        dataIndex: "containerIp",
                                        key: "containerIp"
                                    },
                                    {
                                        title: "HostIP",
                                        dataIndex: "hostIp",
                                        key: "hostIp"
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-name" />,
                                        width: 128,
                                        dataIndex: "name",
                                        key: "name"
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-status" />,
                                        width: 128,
                                        dataIndex: "jobStatus",
                                        key: "jobStatus",
                                        render: (_, r) => <JobStatus status={r.jobStatus}/>
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-creation-time" />,
                                        dataIndex: "createTime",
                                        key: "createTime"
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-startup-time" />,
                                        dataIndex: "startTime",
                                        key: "startTime"
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-end-time" />,
                                        dataIndex: "endTime",
                                        key: "endTime"
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-execution-time" />,
                                        dataIndex: "durationTime",
                                        key: "durationTime",
                                        render:(text)=> <Fragment>{ text && text.split(".")[0] }</Fragment>
                                    },
                                    {
                                        title: <FormattedMessage id="kubedl-dashboard-operation" />,
                                        dataIndex: "options",
                                        render: (_, r) => (
                                            <>
                                                <a onClick={() => this.onLog(r)}><FormattedMessage id="kubedl-dashboard-logs" /></a>
                                            </>
                                        )
                                    }
                                ]}
                                dataSource={detail.specs}
                            />
                        </Card>
                    )}
                    {this.state.tabActiveKey === "events" && (
                        <Card loading={eventsLoading}>
                            {events.length === 0
                            ? <Empty description={<FormattedMessage id="kubedl-dashboard-no-events" />}/>
                            : <div style={{minHeight: 256}}>
                                <LazyLog
                                extraLines={1}
                                enableSearch
                                text={events.join('\n')}
                                caseInsensitive
                                />
                              </div>
                            }
                        </Card>
                    )}
                </div>
                {this.state.logModalVisible && (
                    <LogModal
                        pod={this.state.currentPod}
                        job={detail}
                        onCancel={() => this.onLogClose()}
                    />
                )}
            </PageHeaderWrapper>
        );
    }
}

export default JobDetail


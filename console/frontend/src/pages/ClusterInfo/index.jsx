import {
    ProfileOutlined,
    FundTwoTone
} from "@ant-design/icons";
import {
    Card,
    Row,
    Col,
    Button,
    Tooltip,
    Radio,
    Progress,
    Avatar
} from "antd";
import React, { useState, useRef, useEffect } from "react";
import { history, useIntl } from 'umi';
import { connect } from "dva";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import ProTable from "@ant-design/pro-table";
import {getOverviewNodeInfos, getOverviewTotal, getOverviewRequestPodPhase, getTimeStatistics, getTopResourcesStatistics} from "@/pages/ClusterInfo/service";
import styles from "./style.less";
import { KubeDLIconFont } from "@/utils/iconfont";
import moment from "moment";
const ClusterInfo = ({ globalConfig }) => {
    const intl = useIntl();
    const [loading, setLoading] = useState(true);
    const [statisticsLoading, setStatisticsLoading] = useState(true);
    const [topResourcesLoading, setTopResourcesLoading] = useState(true);
    const [nodeInfos, setNodeInfos] = useState([]);
    const [total, setTotal] = useState(0);
    const [overviewTotal, setOverviewTotal] = useState({
        totalCPU: 0,
        totalMemory: 0,
        totalGPU: 0
    });
    const [overviewRequestPodPhase, setOverviewRequestPodPhase] = useState({
        requestCPU: 0,
        requestMemory: 0,
        requestGPU: 0
    });
    const [dateStatistical, setDateStatistical] = useState('past7days');
    const [statisticalInfos, setStatisticalInfos] = useState([]);
    const [topResourcesInfos, setTopResourcesInfos] = useState([]);
    const [taskTotal, setTaskTotal] = useState(0);
    const pageSizeRef = useRef(20);
    const currentRef = useRef(1);
    const paramsRef = useRef({});
    const fetchIntervalRef = useRef();
    const StatisticsOptions = [
        {
            key: 'past7days',
            label: () => intl.formatMessage({id: 'kubedl-dashboard-past7days'})
        },
        {
            key: 'past30days',
            label: () => intl.formatMessage({id: 'kubedl-dashboard-past30days'})
        },
        {
            key: 'week',
            label: () => intl.formatMessage({id: 'kubedl-dashboard-week'})
        },
        {
            key: 'month',
            label: () => intl.formatMessage({id: 'kubedl-dashboard-month'})
        }
    ].map(name => {
        return{
            value: name.key,
            label: name.label()
        }
    });

    useEffect(() => {
        fetchNodeInfos();
        fetchStatistics(dateStatistical);
        fetchTopResourcesStatistics();
        const interval = 600 * 1000;
        fetchIntervalRef.current = setInterval(() => {
            fetchNodeInfosSilently()
        }, interval);
        return () => {
            clearInterval(fetchIntervalRef.current)
        }
    }, []);


    const fetchNodeInfos = async () => {
        setLoading(true)
        await fetchNodeInfosSilently()
        setLoading(false)
    }

    const fetchNodeInfosSilently = async () => {
        let nodes = await getOverviewNodeInfos({
            current_page: currentRef.current,
            page_size: pageSizeRef.current
        });
        let overviewTotal = await getOverviewTotal();
        let overviewRequestPodPhase = await getOverviewRequestPodPhase();
        setNodeInfos(nodes?.data?.items);
        setOverviewTotal(overviewTotal?.data);
        setOverviewRequestPodPhase(overviewRequestPodPhase?.data);
        setTotal(nodes?.total)
    }

    const fetchStatistics = async (dateStatistical) => {
        setStatisticsLoading(true);
        await fetchGetWeekStatistics(dateStatistical);
        setStatisticsLoading(false);
    }

    const fetchGetWeekStatistics = async (dateStatistical) => {
        const statistics = await getTimeStatistics(fetchDate(dateStatistical).startTime, fetchDate(dateStatistical).endTime);
        setStatisticalInfos(statistics?.data && statistics.data.jobStatistics ? statistics.data.jobStatistics.historyJobs : []);
        setTaskTotal(statistics?.data && statistics.data.jobStatistics ? statistics.data.jobStatistics.totalJobCount : 0)
    }

    const fetchDate = (checkDate) => {
        const now = new Date();
        const day = now.getDay() || 7;
        const endTime = moment(new Date())
            .utc()
            .format();
        if (checkDate === 'past7days') {
            return {
                startTime: moment(new Date(new Date().getTime() - (7 * 24 *60*60*1000)))
                    .utc()
                    .format(),
                endTime: endTime
            }
        }else if (checkDate === 'past30days') {
            return {
                startTime: moment(new Date(new Date().getTime() - (30 * 24 *60*60*1000)))
                    .utc()
                    .format(),
                endTime: endTime
            }
        }else if (checkDate === 'week') {
            return {
                startTime: moment(new Date(now.getFullYear(), now.getMonth(), now.getDate() + 1 - day))
                    .utc()
                    .format(),
                endTime: endTime
            }
        }else {
            return {
                startTime: moment(new Date(now.getFullYear(), now.getMonth(), 1))
                    .utc()
                    .format(),
                endTime: endTime
            }
        }
    }

    const fetchTopResourcesStatistics = async () => {
        setTopResourcesLoading(true);
        const past30days = moment(new Date(new Date().getTime() - (30 * 24 *60*60*1000)))
            .utc()
            .format();
        const topResourcesList = await getTopResourcesStatistics(past30days, pageSizeRef.current);
        setTopResourcesInfos(topResourcesList?.data && topResourcesList.data.runningJobs ? topResourcesList.data.runningJobs : [])
        setTopResourcesLoading(false);
    }

    const onTableChange = (pagination) => {
        if (pagination) {
            currentRef.current = pagination.current;
            pageSizeRef.current = pagination.pageSize;
            fetchNodeInfos()
        }
    }
    const onStatisticsTableChange = (pagination) => {
        if (pagination) {
            currentRef.current = pagination.current;
            pageSizeRef.current = pagination.pageSize;
            fetchStatistics(dateStatistical);
        }
    }
    const onResourcesTableChange = (pagination) => {
        if (pagination) {
            currentRef.current = pagination.current;
            pageSizeRef.current = pagination.pageSize;
            fetchTopResourcesStatistics();
        }
    }
   
    let nodeNameAndNodeTypeAndGpuType =[{
        title: intl.formatMessage({id: 'kubedl-dashboard-node-name'}),
        dataIndex: "nodeName",
        render: (_, record) => {
            return (
                <>
                    <Tooltip title={record.nodeName}>
                        <a>{record.nodeName}</a>
                    </Tooltip>
                </>
            )
        }
    }];
    if(environment){
        nodeNameAndNodeTypeAndGpuType =[
            ...nodeNameAndNodeTypeAndGpuType,
            {
                title: intl.formatMessage({id: 'kubedl-dashboard-node-type'}),
                dataIndex: "instanceType",
                key: "instanceType"
            },
            {
                title: intl.formatMessage({id: 'kubedl-dashboard-node-gpu-type'}),
                dataIndex: "gpuType",
                key: "gpuType"
            },
        ]
    }
    let nodeOtherInformation = [
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-node-total-cpu'}),
            dataIndex: "nodeCpuResources",
            render: (_, record) => {
                return (
                    <>
                        <div>
                            <span>{Math.floor((record.totalCPU - record.requestCPU)/1000)} / </span>
                            <span>{Math.floor(record.totalCPU/1000)}</span>
                        </div>
                    </>
                )
            }
        },
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-node-total-memory'}),
            dataIndex: "nodeMemoryResources",
            render: (_, record) => {
                return (
                    <>
                        <div>
                            <span>{((record.totalMemory - record.requestMemory)/(1024 * 1024 * 1024)).toFixed(2)} / </span>
                            <span>{(record.totalMemory/(1024 * 1024 * 1024)).toFixed(2)}</span>
                        </div>
                    </>
                )
            }
        },
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-node-total-gpu'}),
            dataIndex: "nodeGpuResources",
            render: (_, record) => {
                return (
                    <>
                        {record.totalGPU > 0 ?
                            <div>
                                <span>{Math.floor((record.totalGPU - record.requestGPU)/1000)} / </span>
                                <span>{Math.floor(record.totalGPU/1000)}</span>
                            </div> : '-'
                        }
                    </>
                )
            }
        },
    ];
    const columns = [
        ...nodeNameAndNodeTypeAndGpuType,
        ...nodeOtherInformation
    ];
    const infoColumns = [
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-user'}),
            dataIndex: "userName",
            key: "userName"
        },
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-job-submission-number'}),
            dataIndex: "jobCount",
            key: "jobCount"
        },
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-ratio'}) + '(%)',
            dataIndex: "jobRatio",
            width: 180,
            render: (_, record) => {
                return (
                    <>
                        {record.jobRatio ?
                            <div className={styles.accounted}>
                                <div className={styles.accountedNum}>
                                    {record.jobRatio}
                                </div>
                                <div className={styles.accountedProgress}>
                                    <Progress percent={record.jobRatio} showInfo={false} />
                                </div>
                            </div> : '-'
                        }
                    </>
                )
            }
        },
    ];
    const topInfoColumns = [
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-name'}),
            dataIndex: "name",
            key: "name"
        },
        {
            title: 'GPU ' + intl.formatMessage({id: 'kubedl-dashboard-ratio'}) + '(%)',
            dataIndex: "gpuAccounted",
            render: (_, record) => {
                const computeGpu = record.jobResource && record.jobResource.totalGPU !== 0 ? (record.jobResource.totalGPU/overviewTotal.totalGPU)*100 : 0;
                const gpuAccount = computeGpu !== 0 ? Number((computeGpu).toFixed(2)) : 0;
                return (
                    <>
                        {record.jobResource ?
                            <div>
                                {gpuAccount}
                            </div> : '-'
                        }
                    </>
                )
            }
        },
        {
            title: 'CPU ' + intl.formatMessage({id: 'kubedl-dashboard-ratio'}) + '(%)',
            dataIndex: "cpuAccounted",
            render: (_, record) => {
                const computeCpu = record.jobResource && record.jobResource.totalCPU !== 0 ? (record.jobResource.totalCPU/overviewTotal.totalCPU)*100 : 0;
                const cpuAccount = computeCpu !== 0 ? Number((computeCpu).toFixed(2)): 0;
                return (
                    <>
                        {record.jobResource ?
                            <div>
                                {cpuAccount}
                            </div> : '-'
                        }
                    </>
                )
            }
        },
        {
            title: intl.formatMessage({id: 'kubedl-dashboard-memory-ratio'}) + '(%)',
            dataIndex: "memoryAccounted",
            render: (_, record) => {
                const computeMemory = record.jobResource && record.jobResource.totalMemory !== 0 ? (record.jobResource.totalMemory/overviewTotal.totalMemory)*100 : 0;
                const memoryAccount = computeMemory !== 0 ? Number((computeMemory).toFixed(2)) : 0;
                return (
                    <>
                        {record.jobResource ?
                            <div>
                                {memoryAccount}
                            </div> : '-'
                        }
                    </>
                )
            }
        },
    ];
    const goJobCreate = () => {
        history.push({
            pathname: `/job-submit`,
            query: {}
        });
    };
    const dateStatisticalChange = v =>{
        setDateStatistical(v.target.value);
        fetchStatistics(v.target.value);
    }
    return (
        <div>
            <Card style={{ marginBottom: 12 }} title={
                <div>
                    <h4>{intl.formatMessage({id: 'kubedl-dashboard-cluster-overview'})} ({intl.formatMessage({id: 'kubedl-dashboard-free'})} / {intl.formatMessage({id: 'kubedl-dashboard-total'})})</h4>

                    {/*<Button type="primary" style={{float: 'right'}}>*/}
                    {/*    {intl.formatMessage({id: 'kubedl-dashboard-check-cluster'})}*/}
                    {/*</Button>*/}
                </div>
            }>
                <div>
                    <div>
                        <Row gutter={[24, 24]}>
                            <Col span={7}>
                                <Avatar
                                    className={styles.ackInfoIcon}
                                    size="small"
                                    icon={<KubeDLIconFont type="iconrenwuliebiao-copy"/>} />
                                {intl.formatMessage({id: 'kubedl-dashboard-cpu'})}：{Math.floor((overviewTotal?.totalCPU - overviewRequestPodPhase?.requestCPU)/1000)} / {Math.floor(overviewTotal?.totalCPU/1000)}
                            </Col>
                            <Col span={7} offset={1}>
                                <Avatar
                                    className={styles.ackInfoIcon}
                                    size="small"
                                    icon={<KubeDLIconFont type="iconmemory"/>} />
                                {intl.formatMessage({id: 'kubedl-dashboard-memory'})}：{Math.floor((overviewTotal?.totalMemory - overviewRequestPodPhase?.requestMemory)/(1024 * 1024 * 1024))} / {Math.floor(overviewTotal?.totalMemory/(1024 * 1024 * 1024))}
                            </Col>
                            <Col span={7} offset={1}>
                                <Avatar
                                    className={styles.ackInfoIcon}
                                    size="small"
                                    icon={<KubeDLIconFont type="iconGPUyunfuwuqi"/>} />
                                {intl.formatMessage({id: 'kubedl-dashboard-gpu'})}：{Math.floor((overviewTotal?.totalGPU - overviewRequestPodPhase?.requestGPU)/1000)} / {Math.floor(overviewTotal?.totalGPU/1000)}
                            </Col>
                        </Row>
                    </div>
                </div>
            </Card>

            <Card style={{ marginBottom: 12 }} title={
                <div>
                    {intl.formatMessage({id: 'kubedl-dashboard-job-overview'})}
                    {
                        environment &&
                        <Button type="primary" style={{float: 'right'}} onClick={goJobCreate}>
                            {intl.formatMessage({id: 'kubedl-dashboard-submit-job'})}
                        </Button>
                    }
                </div>
            }>
                <Row gutter={[24, 24]}>
                    <Col span={environment ? 12: 24}>
                        <Card title={
                            <div>
                                <Radio.Group
                                    options={StatisticsOptions}
                                    onChange={dateStatisticalChange}
                                    value={dateStatistical}
                                    optionType="button"
                                    buttonStyle="solid"
                                />
                                <div style={{float:'right'}}>
                                    <div className={styles.taskInfoAvatarDiv}>
                                        <Avatar
                                            className={styles.taskInfoAvatar}
                                            size="small"
                                            icon={<ProfileOutlined />} />
                                    </div>
                                    <div className={styles.taskInfoTitle}>
                                        <span className={styles.taskInfoMessage}>{intl.formatMessage({id: 'kubedl-dashboard-job-total'})}</span>
                                        <span className={styles.taskInfoTotal}>{taskTotal}</span>
                                    </div>
                                </div>
                            </div>
                        }>
                            {/*<Col span={9} offset={1}>*/}
                            {/*    <div className={styles.taskInfoAvatarDiv} style={{marginLeft: '30px'}}>*/}
                            {/*        <Avatar*/}
                            {/*            className={styles.taskInfoAvatar}*/}
                            {/*            size="large"*/}
                            {/*            icon={<FundTwoTone />} />*/}
                            {/*    </div>*/}
                            {/*    <div className={styles.taskInfoTitle}>*/}
                            {/*        <span>{intl.formatMessage({id: 'kubedl-dashboard-new-number'})}</span>*/}
                            {/*        <div>145</div>*/}
                            {/*    </div>*/}
                            {/*</Col>*/}
                            <ProTable
                                loading={statisticsLoading}
                                dataSource={statisticalInfos}
                                headerTitle=""
                                rowKey="task"
                                columns={infoColumns}
                                onChange={onStatisticsTableChange}
                                pagination={{total: total}}
                                toolBarRender={false}
                                search={false}
                            />
                        </Card>
                    </Col>
                    {
                        environment &&
                        <Col span={12}>
                            <Card title={
                                <div>
                                    {intl.formatMessage({id: 'kubedl-dashboard-running-jobs'})}
                                    <Button type="primary" style={{float: 'right'}} onClick={fetchTopResourcesStatistics}>
                                        {intl.formatMessage({id: 'kubedl-dashboard-refresh'})}
                                    </Button>
                                </div>
                            }>
                                <ProTable
                                    loading={topResourcesLoading}
                                    dataSource={topResourcesInfos}
                                    headerTitle=""
                                    rowKey="resources"
                                    columns={topInfoColumns}
                                    onChange={onResourcesTableChange}
                                    pagination={{total: total}}
                                    toolBarRender={false}
                                    search={false}
                                />
                            </Card>
                        </Col>
                    }
                </Row>
            </Card>

            <Card style={{ marginBottom: 12 }} title={
                <div>
                    {intl.formatMessage({id: 'kubedl-dashboard-node-information'})}
                    <Button type="primary" style={{float: 'right'}} onClick={fetchNodeInfos}>
                        {intl.formatMessage({id: 'kubedl-dashboard-refresh'})}
                    </Button>
                </div>
            }>
                <div>
                    <div>
                        <Row gutter={[24, 24]}>
                            <Col span={24}>
                                <ProTable
                                    loading={loading}
                                    dataSource={nodeInfos}
                                    headerTitle=""
                                    rowKey="info"
                                    columns={columns}
                                    onChange={onTableChange}
                                    pagination={{total: total}}
                                    toolBarRender={false}
                                    search={false}
                                />
                            </Col>
                        </Row>
                    </div>
                </div>
            </Card>

        </div>
    );
};

export default connect(({ global }) => ({
    globalConfig: global.config
}))(ClusterInfo);

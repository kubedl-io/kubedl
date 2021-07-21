import {
    DownOutlined,
    PlusOutlined,
    MinusCircleOutlined,
    ReloadOutlined,
    QuestionCircleTwoTone
} from "@ant-design/icons";
import {
    Card,
    Row,
    Col,
    Button, Tooltip,
} from "antd";
import React, { useState, useRef, useEffect } from "react";
import router from "umi/router";
import { connect } from "dva";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import ProTable from "@ant-design/pro-table";
import {queryJobs} from "@/pages/ConsoleInfo/service";
import moment from "moment";

var path = require("path");
const ConsoleInfo = ({ globalConfig }) => {
    const [loading, setLoading] = useState(true);
    const [jobs, setJobs] = useState([]);
    const [total, setTotal] = useState(0)

    const pageSizeRef = useRef(20);
    const currentRef = useRef(1);
    const paramsRef = useRef({});
    const fetchIntervalRef = useRef();

    const searchInitialParameters = {
        jobStatus: "All",
        submitDateRange: [moment().subtract(30, "days"), moment()],
        current: 1,
        page_size: 20,
    };

    useEffect(() => {
        fetchJobs()
        const interval = 3 * 1000
        fetchIntervalRef.current = setInterval(() => {
            fetchJobsSilently()
        }, interval)
        return () => {
            clearInterval(fetchIntervalRef.current)
        }
    }, [])


    const fetchJobs = async () => {
        setLoading(true)
        await fetchJobsSilently()
        setLoading(false)
    }

    const fetchJobsSilently = async () => {
        let queryParams = {...paramsRef.current}
        if (!paramsRef.current.submitDateRange) {
            queryParams = {
                ...queryParams,
                ...searchInitialParameters
            };
        }
        let jobs = await queryJobs({
            name: queryParams.name,
            // namespace: globalConfig.namespace,
            namespace:sessionStorage.getItem("namespace"),
            status: queryParams.jobStatus === "All" ? undefined : queryParams.jobStatus,
            start_time: moment(queryParams.submitDateRange[0]).hours(0).minutes(0).seconds(0)
                .utc()
                .format(),
            end_time: moment(queryParams.submitDateRange[1]).hours(0).minutes(0).seconds(0).add(1,"days")
                .utc()
                .format(),
            current_page: currentRef.current,
            kind: queryParams.jobType,
            page_size: pageSizeRef.current
        });
        const testData = [
            {
                name: 'cn-hangzhou.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 1,
                gpu: 0
            },
            {
                name: 'cn-beijing.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 1,
                gpu: 1
            },
            {
                name: 'cn-shanghai.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 2,
                gpu: 2
            },
            {
                name: 'cn-shenzhen.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 2,
                gpu: 0
            },
            {
                name: 'cn-hangzhou.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 1,
                gpu: 1
            },
            {
                name: 'cn-hangzhou.192.174.35',
                type: 'ecs.gn5-c8g1.2xlarge',
                config: '8CPU 60GB',
                total: 2,
                gpu: 1
            },
        ];
        // setJobs(jobs.data)
        setJobs(testData);
        setTotal(jobs.total)
    }

    const onTableChange = (pagination) => {
        if (pagination) {
            currentRef.current = pagination.current
            pageSizeRef.current = pagination.pageSize
            fetchJobs()
        }
    }

    const columns = [
        {
            title: "名称",
            dataIndex: "name",
            hideInSearch: true,
            render: (_, record) => {
                return (
                    <>
                        <Tooltip title={record.name}>
                            <a>{record.name}</a>
                        </Tooltip>
                    </>
                )
            }
        },
        {
            title: "型号",
            dataIndex: "type",
            hideInSearch: true
        },
        {
            title: "配置",
            dataIndex: "config",
            hideInSearch: true
        },
        {
            title: "GPU(Total)",
            dataIndex: "total",
            hideInSearch: true
        },
        {
            title: "GPU(已分配)",
            dataIndex: "gpu",
            hideInSearch: true
        }
    ];
    return (
        <div>
            <Card style={{ marginBottom: 12 }} title={
                <div>
                    ACK集群信息
                    <Button type="primary" style={{float: 'right'}}>
                        查看集群
                    </Button>
                </div>
            }>
                <div>
                    <div>
                        <h4>集群信息(已分配 / 总量)：</h4>
                        <Row gutter={[24, 24]}>
                            <Col span={7} offset={1}>
                                CPU: 17 / 24
                            </Col>
                            <Col span={7} offset={1}>
                                Memory(GB)：64 / 96
                            </Col>
                            <Col span={7} offset={1}>
                                GPU：3 / 4
                            </Col>
                        </Row>
                    </div>
                    <div>
                        <h4>节点信息：</h4>
                        <Row gutter={[24, 24]}>
                            <Col span={21} offset={1}>
                                <ProTable
                                    loading={loading}
                                    dataSource={jobs}
                                    headerTitle="任务列表"
                                    rowKey="key"
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
}))(ConsoleInfo);

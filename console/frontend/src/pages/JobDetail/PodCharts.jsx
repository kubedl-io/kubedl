import {
    Card,
    Tabs,
    Spin,
    List
} from "antd";
import React, { useState, useRef, useEffect } from "react";
import {
    G2,
    Chart,
    LineAdvance,
    Axis,
    Tooltip,
} from "bizcharts";
import { useIntl } from 'umi';
import moment from 'moment';
import styles from "./style.less";

const { TabPane } = Tabs;

const scale = {
    y: {
        min: 0,
    },
    x:{
        tickCount: 8, // 定义坐标轴刻度线的条数，默认为 5
    }
};
const label = {
    formatter(text, item, index) {
        return moment(Number(text)*1000).format('YYYY-MM-DD HH:mm:ss')
    },
}
export default function PodCharts(props) {
    const intl = useIntl();
    const [cpuData, setCpuData] = useState([]);
    const [gpuData, setGpuData] = useState([]);
    const [memoryData, setMemoryData] = useState([]);
    const podTabs = [
        {
            tab: "CPU",
            key: "CPU",
            data: cpuData,
        },
        {
            tab: "GPU",
            key: "GPU",
            data: gpuData,
        },
        {
            tab: "Memory",
            key: "Memory",
            data: memoryData,
        },
    ];

    useEffect(() => {
        const data = props?.data ?? [];
        if (props.type === 'CPU') {
            setCpuData(data)
        }else if (props.type === 'GPU') {
            setGpuData(data)
        }else {
            setMemoryData(data)
        }
    }, [
        props.data,
        props.loading
    ]);

    const tabOnChange = v => {
        props.tabChange(v);
    }
    return (
        <div style={{width: '100%', background: 'white', margin: '10px 0', padding: '10px'}}>
            <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'kubedl-dashboard-resource-consumption'})}>
                <Tabs shape="wrapped" onChange={tabOnChange} style={{ padding: "10px" }}>
                    {podTabs.map((tab) => (
                        <TabPane tab={tab.tab} key={tab.key}>

                            <Spin spinning={props.loading} size="large">
                                {tab.data.length > 0 ?
                                    <Chart
                                        scale={scale}
                                        autoFit
                                        height={400}
                                        data={tab.data}
                                        containerStyle={{
                                            border: "1px solid #ddd",
                                            borderTop: "none",
                                            padding: "25px",
                                        }}>
                                        <Axis name="x" label={label}/>
                                        <Tooltip
                                            showTitle={false}
                                        >
                                            {
                                                (title, items) => {
                                                    return (<List
                                                        className={styles.chartsList}
                                                        size="small"
                                                        header={''}
                                                        footer={''}
                                                        bordered={false}
                                                        split={false}
                                                        dataSource={items}
                                                        renderItem={(item) => <List.Item>
                                                            <span className={styles.chartsTooltip} style={{backgroundColor: item.color}}></span>
                                                            <span>{item.data.name}</span>
                                                            <span>{moment(Number(item.data.x)*1000).format('YYYY-MM-DD HH:mm:ss')}</span>
                                                            <span>{item.data.y}</span>
                                                        </List.Item>}
                                                    />);
                                                }
                                            }
                                        </Tooltip>
                                        <LineAdvance
                                            shape="smooth"
                                            point={{ size: 3 }}
                                            position="x*y"
                                            // tooltip={["x*y"]}
                                            color="name"
                                            size={2}
                                        />
                                    </Chart> :
                                    <div className={styles.chartsNoData}>
                                        {intl.formatMessage({id: 'kubedl-dashboard-no-data'})}
                                    </div>
                                }
                            </Spin>
                        </TabPane>
                    ))}
                </Tabs>
            </Card>
        </div>
    );
};

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
    Form, Input, Radio, Select, Divider, Alert,
} from "antd";
import React, { useState, useRef, useEffect } from "react";
import router from "umi/router";
import { connect } from "dva";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import {listPVC} from "@/pages/JobSubmit/service";

var path = require("path");

const CodeConfigCreate = ({ globalConfig }) => {
    const [pvcs, setPvcs] = useState([]);
    const [pvcLoading, setPvcLoading] = useState(true);
    const [form] = Form.useForm();
    const formItemLayout = {
        labelCol: { span: 4 },
        wrapperCol: { span: 20 }
    };
    const fetchPVC = async () => {
        setPvcLoading(true);
        const pvcs = await listPVC('default');
        setPvcs(pvcs.data);
        setPvcLoading(false);
    };
    
    const codeAlert = (
        <Alert
            type="info"
            showIcon
            message={
                <span>
          若git为私有库，需要授权平台读取权限，
          <a
              href="https://cs.console.aliyun.com/#/k8s/storage/pvc/list"
              target="_blank"
          >
            指导文档
          </a>
          ，公匙
          <a
              href="https://help.aliyun.com/document_detail/86545.html"
              target="_blank"
          >
              下载
          </a>
        </span>
            }
        />
    );
    return (
        <div>
            <Form
                form={form}
                {...formItemLayout}
                labelAlign="left"
                shouldUpdate
            >
                {() => (
                    <React.Fragment>
                        <Row gutter={[24, 24]}>
                            <Col span={18} offset={3}>
                                <Card style={{ marginBottom: 12 }} title="新增代码配置">
                                    <Form.Item
                                        name="name"
                                        label="名称"
                                    >
                                        <Input />
                                    </Form.Item>
                                    <Form.Item
                                        name="des"
                                        label="描述"
                                    >
                                        <Input />
                                    </Form.Item>
                                    <Form.Item
                                        label="存储来源"
                                        name={["source", "type"]}
                                    >
                                        <Radio.Group>
                                            <Radio value="git">Git</Radio>
                                            <Radio value="NAS">NAS</Radio>
                                            <Radio value="OSS">OSS</Radio>
                                        </Radio.Group>
                                    </Form.Item>
                                    {form.getFieldValue(["source", "type"]) === 'git' && <div>
                                        <Form.Item
                                            name="giturl"
                                            label="Git地址"
                                        >
                                            <Input />
                                        </Form.Item>
                                        <Form.Item
                                            name="code"
                                            label="代码分支"
                                        >
                                            <Input style={{marginBottom: '10px'}}/>
                                            {codeAlert}
                                        </Form.Item>
                                    </div>}
                                    {form.getFieldValue(["source", "type"]) !== 'git' && <div>
                                        <Form.Item
                                            name="namespace"
                                            label="namespace"
                                        >
                                            <Select>
                                                <Select.Option value={1}>1</Select.Option>
                                                <Select.Option value={2}>2</Select.Option>
                                                <Select.Option value={4}>4</Select.Option>
                                            </Select>
                                        </Form.Item>
                                        <Form.Item
                                            name="pvc"
                                            label="存储卷">
                                            <Select
                                                placeholder="选择存储卷声明"
                                                notFoundContent={<span>无存储卷，请先创建</span>}
                                                dropdownRender={menu => (
                                                    <div>
                                                        {menu}
                                                        <Divider style={{ margin: "4px 0" }} />
                                                        <div style={{ textAlign: "center" }}>
                                                            <a onClick={() => fetchPVC()}>
                                                                <ReloadOutlined /> 重新加载
                                                            </a>
                                                        </div>
                                                    </div>
                                                )}
                                            >
                                                {pvcs.map(pvc => (
                                                    <Select.Option title={pvc} value={pvc}>
                                                        {pvc}
                                                    </Select.Option>
                                                ))}
                                            </Select>
                                        </Form.Item>
                                    </div>}
                                    <Form.Item
                                        name="url"
                                        label="本地存放目录"
                                    >
                                        <Input style={{marginBottom: '10px'}}/>
                                    </Form.Item>
                                </Card>
                            </Col>
                        </Row>
                    </React.Fragment>
                )}
            </Form>
        </div>
    );
};

export default connect(({ global }) => ({
    globalConfig: global.config
}))(CodeConfigCreate);

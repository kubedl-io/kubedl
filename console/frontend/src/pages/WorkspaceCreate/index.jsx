import {
    ReloadOutlined,
} from "@ant-design/icons";
import {
    Card, Row, Col, Form, Input,
    Radio, Select, Divider, Alert,
    Button, message, InputNumber
} from "antd";
import { history, useIntl, getLocale } from 'umi';
import React, { useState, useRef, useEffect } from "react";
import { connect } from "dva";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import {listPVC} from "@/pages/JobSubmit/service";
import {newDatasource, newWorkspace} from './service';
import Qs from "qs";

var path = require("path");
const { Option } = Select;
const DataConfig = ({ globalConfig, namespaces, currentUser }) => {
    const defaultDataPath = "/home/jovyan/work";
    const [pvcs, setPvcs] = useState([]);
    const [pvcLoading, setPvcLoading] = useState(true);
    const [ackLink, setAckLink] = useState('');
    const intl = useIntl();
    const [form] = Form.useForm();

    useEffect(() => {
        fetchPVC();
    },[]);

    const formWorkspaceConfig = {

    };

    const formItemLayout = {
        labelCol: { span: getLocale() === 'zh-CN' ? 4 : 5 },
        wrapperCol: { span: getLocale() === 'zh-CN' ? 20 : 19 }
    };

    const fetchPVC = async () => {
        setPvcLoading(true);
        const pvcs = await listPVC(form.getFieldValue("name"));
        setPvcs(pvcs.data);
        setPvcLoading(false);
    };

    const [isLoading, setIsLoading] = useState(false);
    const handleSubmit = (values) => {
        setIsLoading(true);
        const addValues = {
            username: currentUser.loginId ?? '',
            name: values.name,
            description: values.description,
            pvc_name: values.pvc_name,
            local_path: defaultDataPath,
            storage: values.storage
        };
        newWorkspace(addValues).then(res => {
            message.success(intl.formatMessage({id: 'kubedl-dashboard-add-success'}));
            setIsLoading(false);
            history.push({
                pathname: `/workspaces`,
                query: {}
            });
        }).catch(err => {
            setIsLoading(false);
        });
    };

    const onChangeNamespace = (v) => {
        setPvcs([]);
        fetchPVC();
    }
    
    return (
        <div>
            <Form
                initialValues={formWorkspaceConfig}
                form={form}
                {...formItemLayout}
                labelAlign="left"
                onFinish={handleSubmit}
            >
                <Row gutter={[24, 24]}>
                    <Col span={21} offset={3}>
                        <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'kubedl-dashboard-new-workspace'})}>
                            <Form.Item
                                name="name"
                                required={true}
                                label={intl.formatMessage({id: 'kubedl-dashboard-name'})}
                                rules={[
                                    {
                                        required: true,
                                        message: intl.formatMessage({id: 'kubedl-dashboard-please-enter-name'}),
                                    },
                                    {
                                        pattern: '^[0-9a-zA-Z-]{1,32}$',
                                        message: intl.formatMessage({id: 'kubedl-dashboard-name-rules'})
                                    }
                                ]}
                            >
                                <Input />
                            </Form.Item>

                            <Form.Item
                                name="description"
                                label={intl.formatMessage({id: 'kubedl-dashboard-description'})}
                            >
                                <Input />
                            </Form.Item>
                            <Form.Item
                                name="storage"
                                label={intl.formatMessage({id: 'kubedl-dashboard-storage'})}
                                required={true}
                                initialValue={1}
                            >
                                <InputNumber
                                    min={1}
                                    max={96}
                                    step={1}
                                    precision={0}
                                    style={{ width: "100%" }}/>

                            </Form.Item>
                            {/*<Form.Item label={intl.formatMessage({id: 'kubedl-dashboard-local-paths'})}*/}
                            {/*           required={true}*/}
                            {/*           initialValue={'/home/jovyan/work'}*/}
                            {/*           // name = "localpath" // if want to enable updating the input form, uncomment this */}
                            {/*>*/}

                            {/*    <Input />*/}
                            {/*</Form.Item>*/}
                            <Form.Item wrapperCol={{span: 3, offset: 21}}>
                                <Button type="primary" htmlType="submit" loading={isLoading}>{intl.formatMessage({id: 'kubedl-dashboard-submit'})}</Button>
                            </Form.Item>
                        </Card>
                    </Col>
                </Row>
            </Form>
        </div>
    );
};

export default connect(({ global, user }) => ({
    globalConfig: global.config,
    namespaces: global.namespaces,
    currentUser: user.currentUser
}))(DataConfig);
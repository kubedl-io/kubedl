import {
    ReloadOutlined,
} from "@ant-design/icons";
import {
    Card, Row, Col, Form, Input, 
    Radio, Select, Divider, Alert,
    Button, message
} from "antd";
import { history, useIntl, getLocale } from 'umi';
import React, { useState, useRef, useEffect } from "react";
import { connect } from "dva";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import {listPVC} from "@/pages/JobSubmit/service";
import { newDatasource } from './service';

var path = require("path");
const { Option } = Select;
const DataConfig = ({ globalConfig, namespaces, currentUser }) => {
    const defaultDataPath = "/root/data/";
    const [pvcs, setPvcs] = useState([]);
    const [pvcLoading, setPvcLoading] = useState(true);
    const [ackLink, setAckLink] = useState('');
    const intl = useIntl();
    const [form] = Form.useForm();

    useEffect(() => {
        fetchPVC();
    },[]);

    const formDataConfig = {
        name: '',
        description: '',
        type: '',
        namespace: globalConfig.namespace,
        pvc_name: '',
        local_path: ''
    };

    const formItemLayout = {
        labelCol: { span: getLocale() === 'zh-CN' ? 4 : 5 },
        wrapperCol: { span: getLocale() === 'zh-CN' ? 20 : 19 }
    };

    const fetchPVC = async () => {
        setPvcLoading(true);
        const pvcs = await listPVC(form.getFieldValue("namespace"));
        setPvcs(pvcs.data);
        setPvcLoading(false);
    };

    const [isLoading, setIsLoading] = useState(false);
    const handleSubmit = (values) => {
        setIsLoading(true);
        const addValues = {
            userid: currentUser.loginId ?? '',
            username: currentUser.loginName ?? '',
            name: values.name,
            type: '',
            description: values.description,
            namespace: values.namespace,
            pvc_name: values.pvc_name,
            local_path: defaultDataPath + values.local_path
        };
        newDatasource(addValues).then(res => {
            message.success(intl.formatMessage({id: 'dlc-dashboard-add-success'}));
            setIsLoading(false);
            history.push({
                pathname: `/datasheets`,
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
                initialValues={formDataConfig}
                form={form}
                {...formItemLayout}
                labelAlign="left"
                onFinish={handleSubmit}
            >
                <Row gutter={[24, 24]}>
                    <Col span={18} offset={3}>
                        <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'dlc-dashboard-new-data-config'})}>
                            <Form.Item
                                name="name"
                                required={true}
                                label={intl.formatMessage({id: 'dlc-dashboard-name'})}
                                rules={[
                                    {
                                        required: true,
                                        message: intl.formatMessage({id: 'dlc-dashboard-please-enter-name'}),
                                    },
                                    {
                                        pattern: '^[0-9a-zA-Z-]{1,32}$',
                                        message: intl.formatMessage({id: 'dlc-dashboard-name-rules'})
                                    }
                                ]}
                            >
                                <Input />
                            </Form.Item>
                            <Form.Item
                                name="description"
                                label={intl.formatMessage({id: 'dlc-dashboard-description'})}
                            >
                                <Input />
                            </Form.Item>
                            {/*<Form.Item*/}
                            {/*    name="type"*/}
                            {/*    required={true}*/}
                            {/*    label="存储来源"*/}
                            {/*    rules={[*/}
                            {/*        {*/}
                            {/*            required: true,*/}
                            {/*            message: '请选择存储来源!',*/}
                            {/*        }*/}
                            {/*    ]}*/}
                            {/*>*/}
                            {/*    <Radio.Group>*/}
                            {/*        <Radio value="NAS">NAS</Radio>*/}
                            {/*        <Radio value="OSS">OSS</Radio>*/}
                            {/*    </Radio.Group>*/}
                            {/*</Form.Item>*/}
                            <Form.Item
                                name="namespace"
                                required={true}
                                label={intl.formatMessage({id: 'dlc-dashboard-namespace'})}
                                rules={[
                                    {
                                        required: true,
                                        message: intl.formatMessage({id: 'dlc-dashboard-please-enter-namespace'}),
                                    }
                                ]}
                            >
                                <Select
                                    placeholder=""
                                    onChange={onChangeNamespace}
                                >
                                    {namespaces.length > 0 && namespaces.map((item) => <Option key={item.value} value={item.value}>{item.label}</Option>)}
                                </Select>
                            </Form.Item>
                            <Form.Item
                                name="pvc_name"
                                required={true}
                                label={intl.formatMessage({id: 'dlc-dashboard-persistent-volume-claim'})}
                                rules={[
                                    {
                                        required: true,
                                        message: intl.formatMessage({id: 'dlc-dashboard-please-enter-storage-volume'}),
                                    }
                                ]}>
                                <Select
                                    placeholder=""
                                    notFoundContent={<span>{intl.formatMessage({id: 'dlc-dashboard-no-pvc-prompt'})}</span>}
                                    dropdownRender={menu => (
                                        <div>
                                            {menu}
                                            <Divider style={{ margin: "4px 0" }} />
                                            <div style={{ textAlign: "center" }}>
                                                <a onClick={() => fetchPVC()}>
                                                    <ReloadOutlined /> {intl.formatMessage({id: 'dlc-dashboard-reload'})}
                                                </a>
                                            </div>
                                        </div>
                                    )}
                                >
                                    {(pvcs instanceof Array) && pvcs.map(pvc => (
                                        <Select.Option title={pvc} value={pvc}>
                                            {pvc}
                                        </Select.Option>
                                    ))}
                                </Select>
                            </Form.Item>
                            <Form.Item label={intl.formatMessage({id: 'dlc-dashboard-local-paths'})}>
                                <Form.Item
                                    name="local_path"
                                    noStyle>
                                    <Row gutter={[24, 24]}>
                                        <Col span={3}>
                                            <span style={{
                                                lineHeight: '32px',
                                                marginLeft: '10px'
                                            }}>{defaultDataPath}</span>
                                        </Col>
                                        <Col span={21}><Input/></Col>
                                    </Row>
                                </Form.Item>
                            </Form.Item>
                            <Form.Item wrapperCol={{span: 3, offset: 21}}>
                                <Button type="primary" htmlType="submit" loading={isLoading}>{intl.formatMessage({id: 'dlc-dashboard-submit'})}</Button>
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
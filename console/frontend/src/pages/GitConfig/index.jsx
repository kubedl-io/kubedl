import {
    Card,
    Row,
    Col,
    Button,
    Form, Input, message, Alert,
} from "antd";
import React, { useState} from "react";
import { connect } from "dva";
import { history, useIntl, getLocale } from 'umi';
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import { newGitSource } from './service';

const GitConfig = ({ globalConfig, currentUser }) => {
    const defaultCodePath = '/root/code/';
    const intl = useIntl();
    const [form] = Form.useForm();
    const formGitConfig = {
        name: '',
        description: '',
        code_path: '',
        default_branch: '',
        local_path: ''
    };

    const handleGitUrl = (url) => {
        if (url && url !== '') {
            const index = url.lastIndexOf("\/");
            const subUrl = url.substring(index + 1,url.length);
            if (subUrl.indexOf(".git") != -1) {
                return subUrl.split('.')[0];
            }
            return url.substring(index + 1,url.length);
        }
        return ''
    }

    const formItemLayout = {
        labelCol: { span: getLocale() === 'zh-CN' ? 4 : 5 },
        wrapperCol: { span: getLocale() === 'zh-CN' ? 20 : 19 }
    };
    const [isLoading, setIsLoading] = useState(false);
    const handleSubmit = (values) => {
        setIsLoading(true);
        const addValues = {
            userid: currentUser.loginId ?? '',
            username: currentUser.loginId ?? '',
            name: values.name,
            type: 'git',
            description: values.description,
            code_path: values.code_path,
            default_branch: values.default_branch,
            local_path: values.local_path
        };
        newGitSource(addValues).then(res => {
            message.success(intl.formatMessage({id: 'kubedl-dashboard-add-success'}));
            setIsLoading(false);
            history.push({
                pathname: `/datasheets`,
                query: {}
            });
        }).catch(err => {
            setIsLoading(false);
        });
    };

    const promptAlert = (
        <Alert
            type="info"
            showIcon
            message={
                <span>
                    {intl.formatMessage({id: 'kubedl-dashboard-code-synchronization'})}&nbsp;
                    {form.getFieldValue('local_path') ?  form.getFieldValue('local_path')+"/" : defaultCodePath}
                    {handleGitUrl(form.getFieldValue('code_path')) !== '' && handleGitUrl(form.getFieldValue('code_path'))}
                </span>
            }
        />
    );
   
    return (
        <div>
            <Form
                initialValues={formGitConfig}
                form={form}
                {...formItemLayout}
                onFinish={handleSubmit}
                labelAlign="left">
                {() => (
                    <React.Fragment>
                        <Row gutter={[24, 24]}>
                            <Col span={18} offset={3}>
                               <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'kubedl-dashboard-new-create-code-config'})}>
                                    <Form.Item
                                        required={true}
                                        name="name"
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
                                        ]}>
                                        <Input />
                                    </Form.Item>
                                    <Form.Item
                                        name="description"
                                        label={intl.formatMessage({id: 'kubedl-dashboard-description'})}>
                                        <Input />
                                    </Form.Item>
                                    {/*<Form.Item*/}
                                    {/*    required={true}*/}
                                    {/*    label="存储来源"*/}
                                    {/*    name="type"*/}
                                    {/*    rules={[*/}
                                    {/*        {*/}
                                    {/*            required: true,*/}
                                    {/*            message: '请选择存储来源!',*/}
                                    {/*        }*/}
                                    {/*    ]}>*/}
                                    {/*    <Radio.Group>*/}
                                    {/*        <Radio value="GIT">Git</Radio>*/}
                                    {/*    </Radio.Group>*/}
                                    {/*</Form.Item>*/}
                                    <Form.Item
                                        name="code_path"
                                        required={true}
                                        label={intl.formatMessage({id: 'kubedl-dashboard-git-repository'})}
                                        rules={[
                                            {
                                                required: true,
                                                message: intl.formatMessage({id: 'kubedl-dashboard-please-enter-git-path'}),
                                            }
                                        ]}>
                                        <Input />
                                    </Form.Item>
                                    <Form.Item label={intl.formatMessage({id: 'kubedl-dashboard-default-branch'})} required={true}>
                                        <Form.Item
                                            name="default_branch"
                                            noStyle
                                            rules={[
                                                {
                                                    required: true,
                                                    message: intl.formatMessage({id: 'kubedl-dashboard-please-enter-default-branch'}),
                                                }
                                            ]}>
                                            <Input style={{marginBottom: '10px'}}/>
                                        </Form.Item>
                                       {/*codeAlert*/}
                                    </Form.Item>
                                    <Form.Item label={intl.formatMessage({id: 'kubedl-dashboard-local-paths'})}>
                                        <Form.Item
                                            name="local_path"
                                            noStyle>
                                            <Row gutter={[24, 24]}>
                                                {/*<Col span={3}>*/}
                                                {/*    <span style={{*/}
                                                {/*        lineHeight: '32px',*/}
                                                {/*        marginLeft: '10px'*/}
                                                {/*    }}>{defaultCodePath}</span>*/}
                                                {/*</Col>*/}
                                                <Col span={21}><Input/></Col>
                                            </Row>
                                        </Form.Item>
                                        {promptAlert}
                                    </Form.Item>
                                    <Form.Item wrapperCol={{span: 3, offset: 21}}>
                                        <Button type="primary" htmlType="submit" loading={isLoading}>
                                             {intl.formatMessage({id: 'kubedl-dashboard-submit'})}
                                        </Button>
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

export default connect(({ global, user }) => ({
    globalConfig: global.config,
    currentUser: user.currentUser
}))(GitConfig);

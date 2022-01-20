import {DownOutlined, QuestionCircleTwoTone} from "@ant-design/icons";
import {
    Alert,
    AutoComplete,
    Button,
    Card,
    Col,
    Dropdown,
    Form,
    Input,
    InputNumber,
    Menu,
    message,
    Radio,
    Row,
    Select,
    Switch,
    Tabs
} from "antd";
import React, {useEffect, useState} from "react";
import {connect} from "dva";
import {getCodeSource, getDatasources, getNamespaces, submitJob} from "./service";
import Tooltip from "antd/es/tooltip";
import FooterToolbar from "../JobSubmit/components/FooterToolbar";
import {getLocale, history, useIntl} from 'umi';
import {queryCurrentUser} from "@/services/global";
import * as ComForm from "../../components/Form";
import {
    getCommand,
    getTasks,
    getTensorboard,
    handleCodeSource,
    handleCodeSourceBranch,
    handleDataSource,
    handleKindType,
    handleRequirementsData,
    handleWorkingDir
} from '@/utils/JobSubmit'
import styles from "./style.less";

var path = require("path");
const JobSubmitNew = ({ globalConfig }) => {
    const defaultWorkingDir = '/root/';
    const defaultRelativeCodePath = "code/";
    let tfCPUImages = globalConfig["tf-cpu-images"] || [];
    let tfGPUImages = globalConfig["tf-gpu-images"] || [];
    const tfJobImages = tfCPUImages.concat(tfGPUImages);
    const pyTorchImages = globalConfig["pytorch-gpu-images"] || [];
    const intl = useIntl();
    const namespace = "";
    const [submitLoading, setSubmitLoading] = useState(false);
    const [activeTabKey, setActiveTabKey] = useState("Worker");
    const [nameSpaces, setNameSpaces] = useState([]);
    const [dataSource, setDataSource] = useState([]);
    const [nsDataSource, setNsDataSource] = useState([]);
    const [codeSource, setCodeSource] = useState([]);
    const region = location.hostname.split(".")[2] || "cn-hangzhou";
    const [form] = Form.useForm();
    const [cloneInfo, setCloneInfo] = useState(undefined);
    const [usersInfo, setUsersInfo] = useState({});

    if (sessionStorage.getItem("job")) {
        setCloneInfo(JSON.parse(sessionStorage.getItem("job")));
        sessionStorage.removeItem('job');
    }

    const formInitialTF = cloneInfo ? {
        name: cloneInfo?.metadata?.name + '-copy',
        kind: handleKindType(cloneInfo),
        command: getCommand(cloneInfo),
        topoawareschedule: {
            enabled: cloneInfo?.spec?.gpuTopologyPolicy?.isTopologyAware
        },
        requirements: {
            enabled: handleRequirementsData(cloneInfo).enabled,
            text: handleRequirementsData(cloneInfo).text,
            catalog: handleRequirementsData(cloneInfo).catalog
        },
        tensorboard: {
            enabled: getTensorboard(cloneInfo).enabled,
            logDir: getTensorboard(cloneInfo).logDir
        },
        tasks: getTasks(cloneInfo),
        dataSource: handleDataSource(cloneInfo),
        codeSource: handleCodeSource(cloneInfo),
        codeSourceBranch: handleCodeSourceBranch(cloneInfo),
        workingDir: handleWorkingDir(cloneInfo)
    } : {
        name: "",
        kind: "TFJob",
        command: "",
        topoawareschedule: {
            enabled: false
        },
        requirements: {
            enabled: 'textBox',
            text: "",
            catalog: ""
        },
        tensorboard: {
            enabled: false,
            logDir: ""
        },
        tasks: [
            {
                role: "Worker",
                replicas: 1,
                resource: {
                    gpu: 0,
                    cpu: 4,
                    memory: 8
                },
                image: tfJobImages[0]
            }
        ],
        dataSource: null,
        codeSource: null,
        codeSourceBranch: "",
        workingDir: defaultWorkingDir
    };

    const formInitialPyTorch = {
        ...formInitialTF,
        kind: "PyTorchJob",
        tasks: [
            {
                role: "Worker",
                replicas: 1,
                resource: {
                    gpu: 1,
                    cpu: 4,
                    memory: 8
                },
                image: pyTorchImages[0]
            }
        ],
    }

    useEffect(() => {
        fetchSource();
        fetchUser();
    }, []);

    const fetchSource = async () => {
        const dataSource = await getDatasources();
        const gitSource = await getCodeSource();
        const namespaces = await getNamespaces();
        const newDataSource = [];
        const newGitSource = [];
        const newNameSpaces = [];
        if (dataSource && dataSource.data) {
            for (const key in dataSource.data) {
                newDataSource.push(dataSource.data[key]);
            }
        }
        if (gitSource && gitSource.data) {
            for (const key in gitSource.data) {
                newGitSource.push(gitSource.data[key]);
            }
        }
        if (namespaces && namespaces.data) {
            for (const key in namespaces.data) {
                newNameSpaces.push(namespaces.data[key]);
            }
        }
        setDataSource(newDataSource);
        setNsDataSource(newDataSource)
        setCodeSource(newGitSource);
        setNameSpaces(newNameSpaces);
    };

    const fetchUser = async () => {
        const currenteUsers = await queryCurrentUser();
        const userInfos = currenteUsers.data && currenteUsers.data.loginId ? currenteUsers.data : {};
        setUsersInfo(userInfos);
    }

    const onFormSubmit = async form => {
        const newFormKind = ["TFJob", "TFJobDistributed"].includes(form.kind) ? 'TFJob' : 'PyTorchJob';
        const data = {
            apiVersion: "training.kubedl.io/v1alpha1",
            kind: newFormKind,
            metadata: {
                name: form.name,
                namespace: form.nameSpace,
                annotations: {}
            },
            spec: {}
        };
        if (form.tensorboard.enabled === true ) {
            let config = {
                logDir: form.tensorboard.logDir,
                ingressSpec: {
                    // host: window.location.hostname,
                    pathPrefix: "/tensorboards"
                }
            }
            data.metadata.annotations['kubedl.io/tensorboard-config'] = JSON.stringify(config)
        }

        data.metadata.annotations['kubedl.io/tenancy'] = JSON.stringify({
            tenant: "",
            user: usersInfo.loginId ? usersInfo.loginId : '',
        });

        let volumesSpec = [];
        let volumeMountsSpec = [];
        const findCodeSource = codeSource.filter((c) => c.name === form.codeSource)[0];
        const verifyNamespace ={};
        const dataSourceNameObj ={};
        dataSource.forEach(({name,...other})=>{
            dataSourceNameObj[name] = {...other, name};
        })
        if(form.dataSource && form.dataSource.length >0){
            form.dataSource.forEach(({dataSource})=>{
                 if(dataSourceNameObj[dataSource]){
                     let {name, pvc_name, local_path, namespace} = dataSourceNameObj[dataSource];
                    volumesSpec.push({
                        name: 'data-'+ name,
                        persistentVolumeClaim: {
                            claimName: pvc_name
                        }
                    });
                    volumeMountsSpec.push({
                        name: 'data-'+ name,
                        mountPath:local_path
                    });
                    if(!verifyNamespace[namespace]){
                        verifyNamespace[namespace]=true;
                    }else{
                        verifyNamespace.error=true;// 表示存在相同的 namepsace,
                    }
                    data.metadata.namespace = namespace;
                 }
            })
        }
        if(!verifyNamespace.error && verifyNamespace.error !== undefined){
            message.error(intl.formatMessage({id:"kubedl-dashboard-multiple-data-sources-should-be-in-the-same-Namespace"}));
            return;
        }
        // volumes 去重，避免 git 使用 pv 和挂载 pv 重复导致 pod 无法运行
        let volumesNameMap = {}
        volumesSpec = volumesSpec.filter((v) => {
            if (volumesNameMap[v.name]) {
                return false
            } else {
                volumesNameMap[v.name] = 1
                return true
            }
        });

        let requirementsDir = defaultWorkingDir;
        if (findCodeSource && findCodeSource.type === "git") {
            let gitRepoName = path.basename(findCodeSource['code_path'], path.extname(findCodeSource['code_path']));
            let relativeCodePath = defaultRelativeCodePath + gitRepoName;
            let gitConfig = `{"source": "${findCodeSource['code_path']}",
                              "branch": "${form.codeSourceBranch || findCodeSource['default_branch']}",
                              "aliasName": "${"code-" + findCodeSource['name']}",
                              "relativeCodePath": "${relativeCodePath}"}`
            data.metadata.annotations['kubedl.io/git-sync-config'] = gitConfig

            requirementsDir = path.join(defaultWorkingDir, relativeCodePath);
        }
        let replicaCommand = form.command;
        let replicaEnvs = [];
        if (form.requirements?.enabled === 'textBox' && form.requirements.text !== "") {
            replicaEnvs.push({
                name: "REQUIREMENTS_TEXT",
                value: form.requirements.text.trim().replace(/[\n\r]/g,",")
            });
            replicaCommand = "prepare_kubedl_environment && " + form.command
        } else if (form.requirements?.enabled === 'catalog') {
            if (form.requirements.catalog !== "") {
                requirementsDir = form.requirements.catalog
            }
            replicaEnvs.push({
                name: "REQUIREMENTS_DIR",
                value: requirementsDir
            });
            replicaCommand = "prepare_kubedl_environment && " + form.command
        }


        const replicaSpecs = (task,name)=>{
            return {
                replicas: task.replicas,
                template: {
                    spec: {
                        containers: [
                            {
                                command: ["/bin/sh", "-c", replicaCommand],
                                env: replicaEnvs,
                                image: task.image,
                                workingDir: defaultWorkingDir,
                                imagePullPolicy: "Always",
                                name,
                                resources: {
                                    limits: {
                                        cpu: task.resource.cpu,
                                        memory: task.resource.memory + "Gi",
                                        "nvidia.com/gpu": task.resource.gpu || undefined
                                    }
                                },
                                volumeMounts: volumeMountsSpec
                            }
                        ],
                        volumes: volumesSpec
                    }
                }
            }
        };

        if(newFormKind === 'TFJob'){
            if (form.topoawareschedule?.enabled === true) {
                data.spec.gpuTopologyPolicy = {}
                data.spec.gpuTopologyPolicy["isTopologyAware"] = true
            }
            data.spec.tfReplicaSpecs = {};
            form.tasks.forEach(task => {
                data.spec.tfReplicaSpecs[task.role] = replicaSpecs(task,'tensorflow');
            });
            data.kind = 'TFJob';
        }
        if(newFormKind === 'PyTorchJob'){
            if (form.topoawareschedule?.enabled === true) {
                data.spec.gpuTopologyPolicy = {}
                data.spec.gpuTopologyPolicy["isTopologyAware"] = true
            }
            data.spec.pytorchReplicaSpecs = {};
            form.tasks.forEach(task => {
                data.spec.pytorchReplicaSpecs[task.role] = replicaSpecs(task,'pytorch');
            });
            data.kind = 'PyTorchJob';
        }
        try {
            setSubmitLoading(true);
            let ret = await submitJob(data, newFormKind);
            if (ret.code === "200") {
                history.push("/jobs");
            }
        } finally {
            setSubmitLoading(false);
        }
    };

    const onTaskTabEdit = (targetKey, action, fieldOps) => {
        if (action === "remove") {
            const tasks = form.getFieldValue("tasks");
            const removeIndex = tasks.map(t => t.role).indexOf(targetKey);
            setActiveTabKey(tasks[removeIndex - 1].role);
            fieldOps.remove(removeIndex);
        }
    };

    const onTabChange = key => {
        setActiveTabKey(key);
    };
    const onTaskAdd = (e, fieldOps) => {
        form.validateFields([["tasks"]]).then(() => {
            const role = e.key;
            fieldOps.add({
                role: role,
                command: "",
                replicas: 1,
                resource: {
                    gpu: 0,
                    cpu: 4,
                    memory: 8
                },
                image: tfJobImages[0]
            });
            setActiveTabKey(role);
        });
    };

    const tasksHasRole = role => {
        const tasks = form.getFieldValue("tasks");
        if(tasks!==undefined){
            return tasks.some(t => t.role === role);
        }
    };
    const changeTaskType = value => {
        setActiveTabKey("Worker");
        const currentFormInitial = form.getFieldsValue() || {};
        currentFormInitial.kind = value;
        if(["TFJob", "TFJobDistributed"].includes(value)){// 选择TF单击和TF发布
            currentFormInitial.tasks = formInitialTF.tasks;
            form.setFieldsValue(currentFormInitial);
        }else{
            currentFormInitial.tasks = formInitialPyTorch.tasks;
            form.setFieldsValue(currentFormInitial);
        }
    };

    const formItemLayout = {
        labelCol: { span: getLocale() === 'zh-CN' ? 4 : 8 },
        wrapperCol: { span: getLocale() === 'zh-CN' ? 20 : 16 }
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

    const addTaskType = (fieldOps)=> (
        <Dropdown overlay={
            <Menu onClick={e => onTaskAdd(e, fieldOps)}>
               <Menu.Item
                    key="Worker"
                    disabled={tasksHasRole("Worker")}>
                    Worker
               </Menu.Item>
               <Menu.Item key="PS" disabled={tasksHasRole("PS")}>
                   PS
                </Menu.Item>
                <Menu.Item
                    key="Chief"
                    disabled={tasksHasRole("Chief")}>
                    Chief
                </Menu.Item>
                <Menu.Item
                    key="Evaluator"
                    disabled={tasksHasRole("Evaluator")}>
                     Evaluator
                </Menu.Item>
                <Menu.Item
                    key="GraphLearn"
                    disabled={tasksHasRole("GraphLearn")}>
                    GraphLearn
                </Menu.Item>
             </Menu>}>
            <Button type="primary">
                {intl.formatMessage({id: 'kubedl-dashboard-add-task-type'})} <DownOutlined />
            </Button>
        </Dropdown>
    );

    const onSelect = (data) => {
        console.log('onSelect', data);
    };

    const textAlert = (
        <Alert type="info" showIcon
            message={
                <span>
                    {intl.formatMessage({id: 'kubedl-dashboard-third-party-list-prompt'})}
        </span>}/>);

    const nameSpaceChange = (v) => {
        console.log(v)
        var list = []
        for(var i in dataSource) {
            if(dataSource[i].namespace === v) {
                list.push(dataSource[i])
            }
        }
        setNsDataSource(list)
    }

    const gitSourceChange = (v) => {
        const currentFormInitial = form.getFieldsValue() || {};
        const currentDefaultBranch = codeSource.filter((c) => c.name === v)[0] || {}; 
        currentFormInitial.codeSourceBranch = v ? currentDefaultBranch.default_branch : "";
        form.setFieldsValue(currentFormInitial);
    }
    return (
        <div>
            <Form
                initialValues={formInitialTF}
                form={form}
                {...formItemLayout}
                onFinish={onFormSubmit}
                labelAlign="left">
                <Row gutter={[24, 24]}>
                    <Col span={13}>
                        <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'kubedl-dashboard-job-submit'})}>
                            <Form.Item
                                name="name"
                                label={intl.formatMessage({id: 'kubedl-dashboard-job-name'})}
                                rules={[
                                    { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-job-name-required'})},
                                    {
                                        pattern: /^[a-z][-a-z0-9]{0,28}[a-z0-9]$/,
                                        message: intl.formatMessage({id: 'kubedl-dashboard-job-name-required-rules'})
                                    }
                                ]}
                            >
                                <Input />
                            </Form.Item>
                            <ComForm.FormSel
                                form ={form}
                                {...{name:"kind", label:intl.formatMessage({id: 'kubedl-dashboard-job-type'}), rules:[{ required: true }]}}
                                listOption={[
                                              {label:`TF ${intl.formatMessage({id: 'kubedl-dashboard-stand-alone'})}`, value:"TFJob"},
                                              {label:`TF ${intl.formatMessage({id: 'kubedl-dashboard-distributed'})}`, value:"TFJobDistributed"},
                                              {label:`Pytorch ${intl.formatMessage({id: 'kubedl-dashboard-stand-alone'})}`, value:"PyTorchJob"},
                                              {label:`Pytorch ${intl.formatMessage({id: 'kubedl-dashboard-distributed'})}`, value:"PyTorchJobDistributed"},
                                            ]}
                               onChange={changeTaskType}           
                            />
                            <Form.Item
                                name="nameSpace"
                                required={true}
                                label={intl.formatMessage({id: 'kubedl-dashboard-namespace'})}
                                rules={[
                                    {
                                        required: true,
                                        message: intl.formatMessage({id: 'kubedl-dashboard-please-enter-namespace'}),
                                    }
                                ]}
                            >
                                <Select placeholder="" onChange={nameSpaceChange}>
                                    {nameSpaces.length > 0 && nameSpaces.map((item) => <Select.Option key={item} value={item}>{item}</Select.Option>)}
                                </Select>
                            </Form.Item>
                            <Form.Item
                                required={true}
                                name="command"
                                label={intl.formatMessage({id: 'kubedl-dashboard-execute-command'})}>
                                <Input.TextArea  placeholder={''}/>
                            </Form.Item>
                            <ComForm.FromAddDropDown
                              form={form}
                              fieldCode={"dataSource"}
                              fieldKey={"dataSource"}
                              options={nsDataSource}
                              label={intl.formatMessage({id: 'kubedl-dashboard-data-config'})}
                              colStyle={{ labelCol:{ span: getLocale() === 'zh-CN' ? 4 : 8  }, 
                                          wrapperCol:{ span: getLocale() === 'zh-CN' ? 24 : 16  }
                                        }}
                              messageLable={intl.formatMessage({id: 'kubedl-dashboard-pvc-name'})}                     
                            />
                            <Form.Item
                                shouldUpdate
                                noStyle>
                                {() =>(
                                    <div>
                                        <div className={getLocale() === 'zh-CN' ? styles.gitSourceContainer : styles.gitSourceContainerEn}>
                                            <Form.Item
                                                label= {intl.formatMessage({id: 'kubedl-dashboard-code-config'})}
                                                name="codeSource"
                                            >
                                                <Select
                                                    onChange={gitSourceChange}
                                                    allowClear={true}
                                                >
                                                    {codeSource.map(data => (
                                                        data && 
                                                        <Select.Option title={data.name} value={data.name} key={data.name}>
                                                            {data.name}
                                                        </Select.Option>
                                                    ))}
                                                </Select>
                                            </Form.Item>
                                        </div>
                                        {![null, "", undefined].includes(form.getFieldValue("codeSource")) &&
                                        <Row gutter={[24, 24]}>
                                            <Col span={getLocale() === 'zh-CN' ? 20 : 16} offset={getLocale() === 'zh-CN' ? 4 : 8}>
                                                <Alert
                                                    type="info"
                                                    showIcon
                                                    message={
                                                        <span>
                                                            {intl.formatMessage({id: 'kubedl-dashboard-git-repository'})}
                                                            {': '}
                                                            {
                                                                codeSource.length > 0 &&
                                                                codeSource.some((v) => v.name === form.getFieldValue("codeSource")) &&
                                                                codeSource.filter((v) => v.name === form.getFieldValue("codeSource"))[0]['code_path']
                                                            }
                                                            <br/>
                                                            {intl.formatMessage({id: 'kubedl-dashboard-code-local-directory'})}
                                                            {': '}
                                                            {
                                                                codeSource.length > 0 &&
                                                                codeSource.some((v) => v.name === form.getFieldValue("codeSource")) &&
                                                                codeSource.filter((v) => v.name === form.getFieldValue("codeSource"))[0]['local_path'] + '/' +
                                                                handleGitUrl(codeSource.filter((v) => v.name === form.getFieldValue("codeSource"))[0]['code_path'])
                                                            }
                                                        </span>
                                                    }
                                                />
                                            </Col>
                                        </Row>}
                                        {![null, "", undefined].includes(form.getFieldValue("codeSource")) &&
                                        <React.Fragment>
                                            <Form.Item
                                                label={intl.formatMessage({id: 'kubedl-dashboard-code-branch'})}
                                                name={"codeSourceBranch"}
                                                labelCol={{ span: getLocale() === 'zh-CN' ? 3 : 4 , offset: getLocale() === 'zh-CN' ? 4 : 8 }}
                                                wrapperCol={{ span: getLocale() === 'zh-CN' ? 17 : 12  }}
                                            >
                                                <Input placeholder={''}/>
                                            </Form.Item>
                                        </React.Fragment>}
                                    </div>
                                )}
                            </Form.Item>
                            <Form.Item
                                name="workingDir"
                                label={intl.formatMessage({id: 'kubedl-dashboard-working-dir'})}
                            >
                                <Input
                                    placeholder={defaultWorkingDir}
                                    disabled={true}/>
                            </Form.Item>
                            <Form.Item
                                label={intl.formatMessage({id: 'kubedl-dashboard-third-party-config'})}
                                shouldUpdate>
                                {() => (
                                    <React.Fragment>
                                        <Form.Item
                                            name={["requirements", "enabled"]}>
                                            <Radio.Group>
                                                <Radio value="textBox">{intl.formatMessage({id: 'kubedl-dashboard-third-party-list'})}</Radio>
                                                <Radio value="catalog">{intl.formatMessage({id: 'kubedl-dashboard-third-party-directory'})}</Radio>
                                            </Radio.Group>
                                        </Form.Item>
                                        {form.getFieldValue(["requirements", "enabled"]) === 'textBox' &&
                                        <div>
                                            <Form.Item
                                                name={["requirements", "text"]}>
                                                <Input.TextArea  placeholder={`numpy==1.16.4\nabsl-py==0.11.0`}/>
                                            </Form.Item>
                                            {textAlert}
                                        </div>
                                        }
                                        {form.getFieldValue(["requirements", "enabled"]) === 'catalog' &&
                                        <div>
                                            <Form.Item
                                                name={["requirements", "catalog"]}>
                                                <Input.TextArea  placeholder={defaultWorkingDir + defaultRelativeCodePath}/>
                                            </Form.Item>
                                        </div>
                                        }
                                    </React.Fragment>
                                )}
                            </Form.Item>
                            <Form.Item
                                shouldUpdate
                                noStyle
                            >
                                {() =>
                                    (<Form.Item label="Tensorboard">
                                            <Form.Item
                                                name={["tensorboard", "enabled"]}
                                                valuePropName="checked"
                                            >
                                                <Switch />
                                            </Form.Item>
                                            {form.getFieldValue(["tensorboard", "enabled"]) === true &&
                                            <React.Fragment>
                                                <Form.Item
                                                    label={(
                                                        <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-events-dir-prompt'})} >
                                                            {intl.formatMessage({id: 'kubedl-dashboard-events-dir'})} <QuestionCircleTwoTone twoToneColor="#faad14" />
                                                        </Tooltip>
                                                    )}
                                                    name={["tensorboard", "logDir"]}
                                                    rules={[
                                                        { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-events-dir-rules'})}
                                                    ]}
                                                    labelCol={{ span: 10 }}
                                                    wrapperCol={{ span: 18 }}
                                                >
                                                    <Input placeholder={'/root/data/log/'}/>
                                                </Form.Item>
                                            </React.Fragment>}
                                        </Form.Item>
                                    )}
                            </Form.Item>
                        </Card>
                    </Col>
                    <Col span={11}>
                        <Card title={intl.formatMessage({id: 'kubedl-dashboard-resource-info'})} style={{ marginBottom: 12 }}>
                            <Form.List name="tasks">
                                {(fields, fieldOps) => (
                                    <Tabs
                                        type="editable-card"
                                        hideAdd
                                        activeKey={activeTabKey}
                                        onEdit={(targetKey, action) =>
                                            onTaskTabEdit(targetKey, action, fieldOps)
                                        }
                                        onChange={activeKey => onTabChange(activeKey)}
                                        tabBarExtraContent={['TFJobDistributed'].includes(form.getFieldValue('kind')) ? addTaskType(fieldOps) : null }>
                                        {fields.map((field, idx) => (
                                            // <span>{field.name}/{field.fieldKey}</span>
                                            <Tabs.TabPane
                                                tab={form.getFieldValue("tasks")[idx].role}
                                                key={form.getFieldValue("tasks")[idx].role}
                                                closable={
                                                    form.getFieldValue("tasks")[idx].role !== "Worker"}>
                                                <Form.Item
                                                    name={[field.name, "replicas"]}
                                                    label={intl.formatMessage({id: 'kubedl-dashboard-instances-num'})}
                                                    fieldKey={[field.fieldKey, "replicas"]}
                                                    rules={[{ required: true, message:intl.formatMessage({id: 'kubedl-dashboard-instances-num-required'}) }]}>
                                                    <InputNumber
                                                        min={1}
                                                        step={1}
                                                        precision={0}
                                                        style={{ width: "100%" }}
                                                        disabled={
                                                            form.getFieldValue("tasks")[idx].role === "Chief" ||
                                                            ["TFJob", "PyTorchJob"].includes(form.getFieldValue("kind"))
                                                        }/>
                                                </Form.Item>
                                                <Form.Item
                                                    name={[field.name, "image"]}
                                                    label={intl.formatMessage({id: 'kubedl-dashboard-image'})}
                                                    fieldKey={[field.fieldKey, "image"]}
                                                    required={true}>
                                                        {['TFJobDistributed', 'TFJob'].includes(form.getFieldValue('kind'))
                                                            ? <AutoComplete dataSource={tfJobImages}/>
                                                            : <AutoComplete dataSource={pyTorchImages}/>
                                                        }
                                                </Form.Item>
                                                <Form.Item
                                                    name={[field.name, "resource", "cpu"]}
                                                    label={intl.formatMessage({id: 'kubedl-dashboard-cpu'})}
                                                    fieldKey={[field.fieldKey, "resource", "cpu"]}>
                                                    <InputNumber
                                                        min={1}
                                                        max={96}
                                                        step={1}
                                                        precision={0}
                                                        style={{ width: "100%" }}/>
                                                </Form.Item>
                                                <Form.Item
                                                    name={[field.name, "resource", "memory"]}
                                                    label={intl.formatMessage({id: 'kubedl-dashboard-memory'})}
                                                    fieldKey={[field.fieldKey, "resource", "memory"]}>
                                                    <Select>
                                                        <Select.Option value={1}>1GB</Select.Option>
                                                        <Select.Option value={2}>2GB</Select.Option>
                                                        <Select.Option value={4}>4GB</Select.Option>
                                                        <Select.Option value={8}>8GB</Select.Option>
                                                        <Select.Option value={16}>16GB</Select.Option>
                                                        <Select.Option value={32}>32GB</Select.Option>
                                                        <Select.Option value={64}>64GB</Select.Option>
                                                        <Select.Option value={128}>128GB</Select.Option>
                                                        <Select.Option value={256}>256GB</Select.Option>
                                                    </Select>
                                                </Form.Item>
                                                <Form.Item
                                                    name={[field.name, "resource", "gpu"]}
                                                    label={intl.formatMessage({id: 'kubedl-dashboard-gpu'})}
                                                    fieldKey={[field.fieldKey, "resource", "gpu"]}
                                                    dependencies={["tasks"]}
                                                    shouldUpdate={true}>
                                                    <InputNumber
                                                        min={0}
                                                        max={8}
                                                        step={1}
                                                        precision={0}
                                                        style={{ width: "100%" }}
                                                    />
                                                </Form.Item>
                                            </Tabs.TabPane>
                                        ))}
                                    </Tabs>
                                )}
                            </Form.List>
                        </Card>
                        <Card title={intl.formatMessage({id: 'kubedl-dashboard-current-resources-details'})} style={{ marginBottom: 12 }}>
                            {/*<h3>{intl.formatMessage({id: 'kubedl-dashboard-resources-details'})}</h3>*/}
                            <h3 style={{textAlign: 'center'}}>{intl.formatMessage({id: 'kubedl-dashboard-coming-soon'})}</h3>
                        </Card>
                    </Col>
                </Row>
                 <FooterToolbar>
                    <Button type="primary" htmlType="submit">
                        {intl.formatMessage({id: 'kubedl-dashboard-submit-job'})}
                    </Button>
                </FooterToolbar>
            </Form>
        </div>
    );
};

export default connect(({ global }) => ({
    globalConfig: global.config
}))(JobSubmitNew);
import {
  DownOutlined,
  PlusOutlined,
  MinusCircleOutlined,
  ReloadOutlined,
  QuestionCircleTwoTone
} from "@ant-design/icons";
import {
  Button,
  Divider,
  Alert,
  Dropdown,
  Menu,
  Select,
  Tabs,
  Radio,
  Card,
  Row,
  Col,
  Form,
  Input,
  InputNumber,
  Switch
} from "antd";
import React, { useState, useRef, useEffect } from "react";
import { connect, history, useIntl, getLocale } from "umi";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import FooterToolbar from "./components/FooterToolbar";
import { listPVC, submitJob } from "./service";
import Tooltip from "antd/es/tooltip";
import { queryCurrentUser } from "@/services/global";
import { isGit, getCommand, getRequirementsDir, getInfoPvc, getTensorboard, getTasks, getEnv, getMount, getCookie } from '@/utils/JobSubmit'

var path = require("path");

const JobSubmit = ({ globalConfig }) => {
  const intl = useIntl();
  const defaultWorkingDir = '/workspace';
  let tfCPUImages = globalConfig["tf-cpu-images"];
  let tfGPUImages = globalConfig["tf-gpu-images"];
  const tfJobImages = tfCPUImages.concat(tfGPUImages);
  const pyTorchImages = globalConfig["pytorch-gpu-images"];

  const namespace = globalConfig["namespace"];
  const [pvcs, setPvcs] = useState([]);
  const [pvcLoading, setPvcLoading] = useState(true);
  const [submitLoading, setSubmitLoading] = useState(false);
  const [activeTabKey, setActiveTabKey] = useState("Worker");
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
      kind: cloneInfo?.kind,
      command: getCommand(cloneInfo),
      source: {
          type: isGit(cloneInfo) ? 'git' : 'pvc',
          gitRepo: isGit(cloneInfo) ? eval('('+ cloneInfo.metadata.annotations['kubedl.io/git-sync-config']+')')?.source :"",
          gitBranch: isGit(cloneInfo) ? eval('('+ cloneInfo.metadata.annotations['kubedl.io/git-sync-config']+')')?.branch :"",
          pvc: isGit(cloneInfo) ? undefined : getInfoPvc(cloneInfo),
      },
      topoawareschedule: {
          enabled: cloneInfo?.spec?.gpuTopologyPolicy?.isTopologyAware
      },
      requirements: {
          enabled: getRequirementsDir(cloneInfo).enabled,
          requirementsDir: getRequirementsDir(cloneInfo).requirementsDir
      },
      tensorboard: {
          enabled: getTensorboard(cloneInfo).enabled,
          logDir: getTensorboard(cloneInfo).logDir
      },
      tasks: getTasks(cloneInfo),
      env: getEnv(cloneInfo),
      mount: getMount(cloneInfo)
  } : {
    name: "",
    kind: "TFJob",
    command: "",
    source: {
      type: "git",
      gitRepo: "",
      gitBranch: "",
      pvc: undefined
    },
    topoawareschedule: {
      enabled: false
    },
    requirements: {
      enabled: false,
      requirementsDir: ""
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
    env: [],
    mount: []
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
    fetchPVC();
    fetchUser();
  }, []);

  const fetchPVC = async () => {
    setPvcLoading(true);
    const pvcs = await listPVC(globalConfig.namespace);
    setPvcs(pvcs.data);
    setPvcLoading(false);
  };

  const fetchUser = async () => {
      const currenteUsers = await queryCurrentUser();
      const userInfos = currenteUsers.data && currenteUsers.data.loginId ? currenteUsers.data : {};
      setUsersInfo(userInfos);
  }

  const onFormSubmit = async form => {
    const data = {
      apiVersion: "kubeflow.org/v1",
      kind: "TFJob",
      metadata: {
        name: form.name,
        namespace: globalConfig.namespace,
        annotations: {}
      },
      spec: {}
    };

    if (form.tensorboard.enabled === true ) {
      let config = {
        logDir: form.tensorboard.logDir,
        ingressSpec: {
          // host: window.location.hostname,
          pathPrefix: "/"
        }
      }
      data.metadata.annotations['kubedl.io/tensorboard-config'] = JSON.stringify(config)
    }

    data.metadata.annotations['kubeflow.org/tenancy'] = JSON.stringify({
        tenant: "",
        user: usersInfo.loginId ? usersInfo.loginId : '',
    });

    let volumesSpec = (form.mount || []).map(m => ({
      name: m.pvc,
      persistentVolumeClaim: {
        claimName: m.pvc
      }
    }));

    const volumeMountsSpec = (form.mount || []).map(m => ({
      name: m.pvc,
      mountPath: m.path
    }));

    if (form.source.type === 'pvc') {
      volumesSpec.push({
        name: form.source.pvc,
        persistentVolumeClaim: {
          claimName: form.source.pvc
        }
      });
      volumeMountsSpec.push({
        name: form.source.pvc,
        mountPath: defaultWorkingDir,
      })
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
    })

    let requirementsDir = defaultWorkingDir;
    if (form.source.type === 'git') {
      let gitConfig = `{"source": "${form.source.gitRepo}","branch": "${form.source.gitBranch || 'master'}"}`
      data.metadata.annotations['kubedl.io/git-sync-config'] = gitConfig
      let gitRepoName = path.basename(form.source.gitRepo, path.extname(form.source.gitRepo))
      requirementsDir = path.join(defaultWorkingDir, gitRepoName)
    }

    let replicaCommand = form.command
    let replicaEnvs = form.env
    if (form.requirements.enabled === true) {
      if (form.requirements.requirementsDir === "") {
        replicaEnvs.push({
          name: "REQUIREMENTS_DIR",
          value: requirementsDir
        })
      } else {
        replicaEnvs.push({
          name: "REQUIREMENTS_DIR",
          value: form.requirements.requirementsDir
        })
      }

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

    if(form.kind === 'TFJob'){
      if (form.topoawareschedule.enabled === true) {
        data.spec.gpuTopologyPolicy = {}
        data.spec.gpuTopologyPolicy["isTopologyAware"] = true
      }
      data.spec.tfReplicaSpecs = {};
      form.tasks.forEach(task => {
          data.spec.tfReplicaSpecs[task.role] = replicaSpecs(task,'tensorflow');
      });
      data.kind = 'TFJob';
    }
    if(form.kind === 'PyTorchJob'){
      if (form.topoawareschedule.enabled === true) {
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
      let ret = await submitJob(data, form.kind);
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
      if(value === 'TFJob'){
          form.setFieldsValue(formInitialTF);
      }else{
          form.setFieldsValue(formInitialPyTorch);
      }
  };

  const formItemLayout = {
    labelCol: { span: getLocale() === 'zh-CN' ? 4 : 8 },
    wrapperCol: { span: getLocale() === 'zh-CN' ? 20 : 16 }
  };

  const inlineFormItemLayout = {
    labelCol: { span: 0 },
    wrapperCol: { span: 22 }
  };

  const pvcSelector = (
    <Select
      placeholder={intl.formatMessage({id: 'kubedl-dashboard-pvc-select-prompt'})}
      notFoundContent={<span>{intl.formatMessage({id: 'kubedl-dashboard-no-pvc-prompt'})}</span>}
      dropdownRender={menu => (
        <div>
          {menu}
          <Divider style={{ margin: "4px 0" }} />
          <div style={{ textAlign: "center" }}>
            <a onClick={() => fetchPVC()}>
              <ReloadOutlined /> {intl.formatMessage({id: 'kubedl-dashboard-reload'})}
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
  )
    const addTaskType = (fieldOps)=> (
        <Dropdown
            overlay={
                <Menu onClick={e => onTaskAdd(e, fieldOps)}>
                    <Menu.Item
                        key="Worker"
                        disabled={tasksHasRole("Worker")}
                    >
                        Worker
                    </Menu.Item>
                    <Menu.Item key="PS" disabled={tasksHasRole("PS")}>
                        PS
                    </Menu.Item>
                    <Menu.Item
                        key="Chief"
                        disabled={tasksHasRole("Chief")}
                    >
                        Chief
                    </Menu.Item>
                    <Menu.Item
                        key="Evaluator"
                        disabled={tasksHasRole("Evaluator")}
                    >
                        Evaluator
                    </Menu.Item>
                    <Menu.Item
                        key="GraphLearn"
                        disabled={tasksHasRole("GraphLearn")}
                    >
                        GraphLearn
                    </Menu.Item>
                </Menu>
            }
        >
            <Button type="primary">
              {intl.formatMessage({id: 'kubedl-dashboard-add-task-type'})} <DownOutlined />
            </Button>
        </Dropdown>
    );
  const pvcAlert = (
    <Alert
      type="info"
      showIcon
      message={
        <span>
          {intl.formatMessage({id: 'kubedl-dashboard-pvc-alert-info1'})} Kubernetes {intl.formatMessage({id: 'kubedl-dashboard-create'})}
          <a
            href="https://cs.console.aliyun.com/#/k8s/storage/pvc/list"
            target="_blank"
          >
             &nbsp;{intl.formatMessage({id: 'kubedl-dashboard-pvc-statement'})}&nbsp;
          </a>
          {intl.formatMessage({id: 'kubedl-dashboard-pvc-alert-info2'})}"{namespace}"，{intl.formatMessage({id: 'kubedl-dashboard-please-refer'})}
          <a
            href="https://help.aliyun.com/document_detail/86545.html"
            target="_blank"
          >
             &nbsp;{intl.formatMessage({id: 'kubedl-dashboard-document-links'})}
          </a>
        </span>
      }
    />
  );
  const renderImages = i=>
      <Select.Option title={i} value={i}>
      {i.split(/\/(.+)/)[1]}
      </Select.Option>;
  return (
    <div>
      <Form
        initialValues={formInitialTF}
        form={form}
        {...formItemLayout}
        onFinish={onFormSubmit}
        labelAlign="left"
      >
        <Row gutter={[24, 24]}>
          <Col span={14}>
            <Card style={{ marginBottom: 12 }} title={intl.formatMessage({id: 'kubedl-dashboard-basic-info'})}>
              <Form.Item
                name="kind"
                label={intl.formatMessage({id: 'kubedl-dashboard-job-type'})}
                rules={[{ required: true }]}
              >
                <Select style={{ width: 120 }} onChange={changeTaskType}>
                  <Select.Option value="TFJob">TensorFlow</Select.Option>
                  <Select.Option value="PyTorchJob">PyTorch</Select.Option>
                </Select>
              </Form.Item>
              <Form.Item
                name="name"
                label={intl.formatMessage({id: 'kubedl-dashboard-job-name'})}
                rules={[
                  { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-job-name-required'}) },
                  {
                    pattern: /^[a-z][-a-z0-9]{0,28}[a-z0-9]$/,
                    message: intl.formatMessage({id: 'kubedl-dashboard-job-name-required-rules'})
                  }
                ]}
              >
                <Input />
              </Form.Item>
            </Card>
            <Card title={intl.formatMessage({id: 'kubedl-dashboard-job-overview'})}>
              <Form.Item
                label={(
                  <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-code-mounts-dir'}) + ":" + defaultWorkingDir} >
                    {intl.formatMessage({id: 'kubedl-dashboard-code-config'})} <QuestionCircleTwoTone twoToneColor="#faad14" />
                  </Tooltip>
                )}
                shouldUpdate
                rules={[
                  { required: true }
                ]}
              >
              {() => (
                <React.Fragment>
                  <Form.Item
                    name={["source", "type"]}
                  >
                    <Radio.Group>
                      <Radio value="git">{intl.formatMessage({id: 'kubedl-dashboard-code-repository'})}</Radio>
                      <Radio value="pvc">{intl.formatMessage({id: 'kubedl-dashboard-cloud-storage-mount'})}</Radio>
                    </Radio.Group>
                  </Form.Item>
                  {form.getFieldValue(["source", "type"]) === 'git' &&
                  <div>
                    <Form.Item
                      label={(
                        <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-repository-address-prompt'})} >
                          {intl.formatMessage({id: 'kubedl-dashboard-repository-address'})} <QuestionCircleTwoTone twoToneColor="#faad14" />
                        </Tooltip>
                      )}
                      name={["source", "gitRepo"]}
                      rules={[
                        { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-git-repository-address-required'})}
                      ]}
                      labelCol={{ span: getLocale() === 'zh-CN' ? 6 : 10 }}
                      wrapperCol={{ span: getLocale() === 'zh-CN' ? 18 : 14 }}
                    >
                      <Input placeholder="https://github.com/kubedl"/>
                    </Form.Item>
                    <Form.Item
                      label={intl.formatMessage({id: 'kubedl-dashboard-code-branch'})}
                      name={["source", "gitBranch"]}
                      labelCol={{ span: 6 }}
                      wrapperCol={{ span: 18 }}
                    >
                      <Input placeholder={intl.formatMessage({id: 'kubedl-dashboard-code-branch-prompt'})}/>
                    </Form.Item>
                  </div>
                  }
                  {form.getFieldValue(["source", "type"]) === 'pvc' &&
                  <div>
                    <Form.Item
                      name={["source", "pvc"]}
                      rules={[
                        { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-git-repository-address-required'})}
                      ]}
                    >
                      {pvcSelector}
                    </Form.Item>
                    {pvcAlert}
                  </div>
                  }
                </React.Fragment>
              )}
              </Form.Item>
              <Form.Item
                name="command"
                label={intl.formatMessage({id: 'kubedl-dashboard-execute-command'})}
                rules={[{ required: true, message: intl.formatMessage({id: 'kubedl-dashboard-execute-command-required'})}]}
              >
                <Input.TextArea  placeholder={`python examples/main.py`}/>
              </Form.Item>



              <Form.Item
                shouldUpdate
                noStyle
              >
              {() =>
              (<Form.Item
                label={(
                  <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-third-party-libraries-prompt'})} >
                    {intl.formatMessage({id: 'kubedl-dashboard-third-party-config'})} <QuestionCircleTwoTone twoToneColor="#faad14" />
                  </Tooltip>
                )}
               >
                  <Form.Item
                    name={["requirements", "enabled"]}
                    valuePropName="checked"
                  >
                    <Switch />
                  </Form.Item>
                  {form.getFieldValue(["requirements", "enabled"]) === true &&
                  <React.Fragment>
                    <Form.Item
                      label={(
                        <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-dir-prompt'})}>
                          {intl.formatMessage({id: 'kubedl-dashboard-dir'})} <QuestionCircleTwoTone twoToneColor="#faad14" />
                        </Tooltip>
                      )}
                      name={["requirements", "requirementsDir"]}
                      rules={[
                        {
                          pattern: /^\/\S+$/,
                          message: intl.formatMessage({id: 'kubedl-dashboard-dir-rules'})
                        }
                      ]}
                      labelCol={{ span: 6 }}
                      wrapperCol={{ span: 18 }}
                    >
                      <Input placeholder={intl.formatMessage({id: 'kubedl-dashboard-dir-placeholder'})}/>
                    </Form.Item>
                  </React.Fragment>}
                </Form.Item>
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
                      labelCol={{ span: 6 }}
                      wrapperCol={{ span: 18 }}
                    >
                      <Input placeholder={"/root/data/log/"}/>
                    </Form.Item>
                  </React.Fragment>}
                </Form.Item>
              )}
              </Form.Item>
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
                    tabBarExtraContent={ form.getFieldValue('kind')==='TFJob' ? addTaskType(fieldOps) : null }
                  >
                    {fields.map((field, idx) => (
                      // <span>{field.name}/{field.fieldKey}</span>
                      <Tabs.TabPane
                        tab={form.getFieldValue("tasks")[idx].role}
                        key={form.getFieldValue("tasks")[idx].role}
                        closable={
                          form.getFieldValue("tasks")[idx].role !== "Worker"
                        }
                      >
                        <Form.Item
                          name={[field.name, "replicas"]}
                          label={intl.formatMessage({id: 'kubedl-dashboard-instances-num'})}
                          fieldKey={[field.fieldKey, "replicas"]}
                          rules={[{ required: true, message: intl.formatMessage({id: 'kubedl-dashboard-instances-num-required'}) }]}
                        >
                          <InputNumber
                            min={1}
                            step={1}
                            precision={0}
                            style={{ width: "100%" }}
                            disabled={
                              form.getFieldValue("tasks")[idx].role === "Chief"
                            }
                          />
                        </Form.Item>
                        <Form.Item
                          name={[field.name, "image"]}
                          label={intl.formatMessage({id: 'kubedl-dashboard-image'})}
                          fieldKey={[field.fieldKey, "image"]}
                          required={true}
                        >
                          <Select>
                            {form.getFieldValue('kind')==='TFJob'
                                ? tfJobImages.map(i => renderImages(i))
                                : pyTorchImages.map(i => renderImages(i))}
                          </Select>
                        </Form.Item>
                        <Form.Item
                          name={[field.name, "resource", "cpu"]}
                          label={intl.formatMessage({id: 'kubedl-dashboard-cpu'})}
                          fieldKey={[field.fieldKey, "resource", "cpu"]}
                        >
                          <InputNumber
                            min={1}
                            max={96}
                            step={1}
                            precision={0}
                            style={{ width: "100%" }}
                          />
                        </Form.Item>
                        <Form.Item
                          name={[field.name, "resource", "memory"]}
                          label={intl.formatMessage({id: 'kubedl-dashboard-memory'})}
                          fieldKey={[field.fieldKey, "resource", "memory"]}
                        >
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
                          shouldUpdate={true}
                        >
                          {/* {() => {
                            console.log(123);
                            return ( */}
                            <InputNumber
                              min={0}
                              max={8}
                              step={1}
                              precision={0}
                              style={{ width: "100%" }}
                            />
                            {/* )
                          }} */}
                        </Form.Item>
                      </Tabs.TabPane>
                    ))}
                  </Tabs>
                )}
              </Form.List>
            </Card>
          </Col>
          <Col span={10}>
            <Card title={intl.formatMessage({id: 'kubedl-dashboard-environment-variable'})} style={{ marginBottom: 12 }}>
              <Form.List name="env">
                {(fields, fieldOps) => (
                  <div>
                    {fields.map((field, idx) => (
                      <div key={idx}>
                        <Form.Item
                          {...inlineFormItemLayout}
                          name={[field.name, "name"]}
                          fieldKey={[field.fieldKey, "name"]}
                          validateTrigger={["onChange", "onBlur"]}
                          style={{ display: "inline-block" }}
                          rules={[
                            { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-environment-variable-required'}) },
                            {
                              pattern: /^[-._a-zA-Z][-._a-zA-Z0-9]*$/,
                              message: intl.formatMessage({id: 'kubedl-dashboard-environment-variable-rules'})
                            }
                          ]}
                        >
                          <Input placeholder={intl.formatMessage({id: 'kubedl-dashboard-environment-variable-name'})} />
                        </Form.Item>
                        <Form.Item
                          {...inlineFormItemLayout}
                          name={[field.name, "value"]}
                          fieldKey={[field.fieldKey, "value"]}
                          validateTrigger={["onChange", "onBlur"]}
                          style={{ display: "inline-block" }}
                          rules={[
                            { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-environment-variable-value-required'}) }
                          ]}
                        >
                          <Input placeholder={intl.formatMessage({id: 'kubedl-dashboard-environment-variable-value'})} />
                        </Form.Item>
                        <MinusCircleOutlined
                          className="dynamic-delete-button"
                          style={{ display: "inline-block", margin: "8px" }}
                          onClick={() => {
                            fieldOps.remove(idx);
                          }}
                        />
                      </div>
                    ))}

                    <Form.Item>
                      <Button
                        type="primary"
                        ghost
                        onClick={() => {
                          fieldOps.add({
                            name: "",
                            value: ""
                          });
                        }}
                        style={{ width: "80%" }}
                      >
                        <PlusOutlined /> {intl.formatMessage({id: 'kubedl-dashboard-environment-variable-add'})}
                      </Button>
                    </Form.Item>
                  </div>
                )}
              </Form.List>
            </Card>

            <Card title={intl.formatMessage({id: 'kubedl-dashboard-storage-config'})}>
              <Form.List name="mount">
                {(fields, fieldOps) => (
                  <div>
                    {fields.map((field, idx) => (
                      <div key={idx}>
                        <Form.Item
                          {...inlineFormItemLayout}
                          name={[field.name, "pvc"]}
                          fieldKey={[field.fieldKey, "pvc"]}
                          validateTrigger={["onChange", "onBlur"]}
                          style={{ display: "inline-block", width: "45%" }}
                          rules={[
                            { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-pvc-statement-required'}) },
                          ]}
                        >
                          {pvcSelector}
                        </Form.Item>
                        <Form.Item
                          {...inlineFormItemLayout}
                          name={[field.name, "path"]}
                          fieldKey={[field.fieldKey, "path"]}
                          validateTrigger={["onChange", "onBlur"]}
                          style={{ display: "inline-block", width: "45%" }}
                          rules={[
                            { required: true, message: intl.formatMessage({id: 'kubedl-dashboard-mount-path-required'}) },
                            {
                              pattern: /^\/\S+$/,
                              message: intl.formatMessage({id: 'kubedl-dashboard-mount-path-rules'})
                            },
                            { validator: async (rule, value) => {
                                if (value === defaultWorkingDir || value === defaultWorkingDir + '/') {
                                  throw new Error(intl.formatMessage({id: 'kubedl-dashboard-mount-path-rules-error'}))
                                }
                              }
                            }
                          ]}
                        >
                          <Input placeholder={intl.formatMessage({id: 'kubedl-dashboard-mount-path'})} />
                        </Form.Item>
                        <MinusCircleOutlined
                          className="dynamic-delete-button"
                          style={{ display: "inline-block", margin: "8px" }}
                          onClick={() => {
                            fieldOps.remove(idx);
                          }}
                        />
                      </div>
                    ))}

                    <Form.Item>
                      <Button
                        type="primary"
                        ghost
                        onClick={() => {
                          fieldOps.add({
                            path: "",
                            pvc: undefined,
                          });
                        }}
                        style={{ width: "80%" }}
                      >
                        <PlusOutlined /> {intl.formatMessage({id: 'kubedl-dashboard-storage-config-add'})}
                      </Button>
                    </Form.Item>
                    {pvcAlert}
                  </div>
                )}
              </Form.List>
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

export default connect(({ global, jobSubmit }) => ({
  globalConfig: global.config
}))(JobSubmit);
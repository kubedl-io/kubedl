import React, {useEffect, useRef, useState} from "react";
import {Button, Descriptions, Divider, Form, Input, message, Modal, Spin, Tooltip,} from "antd";
import {CheckOutlined, EditOutlined, QuestionCircleTwoTone,} from "@ant-design/icons";
import PodStatus from "@/components/PodStatus";
import {getJobTensorboardStatus, reApplyJobTensorboard} from "./service";
import {useIntl} from "umi";

const CreateTBModal = ({ onCancel, selectedJob = null, isViewing = false }) => {
  const intl = useIntl();
  const [createForm] = Form.useForm();
  const [updateForm] = Form.useForm();

  const [isCreating, setIsCreating] = useState(true);
  const [tbLoading, setTbLoading] = useState(false);
  const [tbStatus, setTbStatus] = useState(null);
  const [isEditing, setIsEditing] = useState(false);
  const refreshInterval = useRef(null);

  useEffect(() => {
    if (isViewing) {
      fetchTensorboard();
      setIsCreating(false);
    }
    return () => stopFetchTensorboardLoop();
  }, []);

  const fetchTensorboard = async () => {
    setTbLoading(true);
    await fetchTensorboardSilently();
    setTbLoading(false);
    startFetchTensorboardLoop();
  };

  const startFetchTensorboardLoop = () => {
    refreshInterval.current = setInterval(() => {
      fetchTensorboardSilently();
    }, 2500);
  };

  const stopFetchTensorboardLoop = () => {
    clearInterval(refreshInterval.current);
  };

  const fetchTensorboardSilently = async () => {
    let res = await getJobTensorboardStatus({
      job_namespace: selectedJob.namespace,
      job_name: selectedJob.name,
      job_uid: selectedJob.id,
      kind: selectedJob.jobType,
    });
    setTbStatus(res.data);
  };

  const onSubmit = async (form) => {
    await reApplyJobTensorboard(
      {
        job_namespace: selectedJob.namespace,
        job_name: selectedJob.name,
        job_uid: selectedJob.id,
        kind: selectedJob.jobType,
      },
      {
        logDir: form.logDir,
        ingressSpec: {
          // host: window.location.hostname,
          pathPrefix: "/tensorboards",
        },
      }
    );
    fetchTensorboard();
    setIsCreating(false);
  };

  const onEditConfig = () => {
    stopFetchTensorboardLoop();
    setIsEditing(true);
  };

  const onUpdateConfig = async () => {
    updateForm.validateFields().then(async (values) => {
      let logDir = values.logDir;
      await reApplyJobTensorboard(
        {
          job_namespace: selectedJob.namespace,
          job_name: selectedJob.name,
          job_uid: selectedJob.id,
          kind: selectedJob.jobType,
        },
        {
          logDir: logDir,
          ingressSpec: {
            // host: window.location.hostname,
            pathPrefix: "/tensorboards",
          },
        }
      );
      message.success(
        intl.formatMessage({ id: "kubedl-dashboard-update-log-success" })
      );
      fetchTensorboard();
      setIsEditing(false);
    });
  };

  let CreateForm = () => (
    <Form form={createForm} layout="vertical">
      <Form.Item
        label={
          <Tooltip
            title={
              "Job " +
              intl.formatMessage({ id: "kubedl-dashboard-events-dir-prompt" })
            }
          >
            {intl.formatMessage({ id: "kubedl-dashboard-events-dir" })}{" "}
            <QuestionCircleTwoTone twoToneColor="#faad14" />
          </Tooltip>
        }
        name={["logDir"]}
        rules={[
          {
            required: true,
            message: intl.formatMessage({
              id: "kubedl-dashboard-events-dir-required",
            }),
          },
        ]}
        labelCol={{ span: 10 }}
        wrapperCol={{ span: 18 }}
      >
        <Input placeholder={"/root/data/"} />
      </Form.Item>
    </Form>
  );

  let ViewContainer = () => {

    let ingress = tbStatus?.ingresses?.[0]?.status?.loadBalancer?.ingress?.[0]
    let fullUrl
    let job_namespace = selectedJob.namespace;
    let job_name = selectedJob.name;
    if (ingress) {
      if (ingress.ip) {
        fullUrl = `http://${ingress.ip}/tensorboards/${job_namespace}/${job_name}`
      } else if (ingress.hostname) {
        fullUrl = `http://${ingress.hostname}/tensorboards/${job_namespace}/${job_name}`
      }
    }
    return (
        <div>
          {tbLoading ? (
              <div style={{textAlign: "center", padding: "30px"}}>
                <Spin/>
              </div>
          ) : (
              <div>
                <Descriptions size="small">
                  <Descriptions.Item
                      label={intl.formatMessage({id: "kubedl-dashboard-domain-name"})}
                      span={3}
                  >
                    {tbStatus.ingresses ? (
                        <a
                            target="_blank"
                            href={fullUrl}
                        >
                          {intl.formatMessage({id: "kubedl-dashboard-open"})} Tensorboard
                        </a>
                    ) : (
                        "-"
                    )}
                  </Descriptions.Item>
                  <Descriptions.Item
                      label={intl.formatMessage({id: "kubedl-dashboard-status"})}
                      span={3}
                  >
                    <PodStatus status={tbStatus.phase}></PodStatus>
                  </Descriptions.Item>
                  <Descriptions.Item
                      label={intl.formatMessage({id: "kubedl-dashboard-details"})}
                      span={3}
                  >
                    {tbStatus.message || "-"}
                  </Descriptions.Item>
                </Descriptions>
                <Divider/>
                <Descriptions size="small">
                  <Descriptions.Item
                      label={intl.formatMessage({id: "kubedl-dashboard-events-dir"})}
                      span={3}
                  >
                    {isEditing ? (
                        <div>
                          <Form form={updateForm}>
                            <Form.Item
                                label={null}
                                name={["logDir"]}
                                initialValue={tbStatus?.config?.logDir ?? null}
                                rules={[
                                  {
                                    required: true,
                                    message: intl.formatMessage({
                                      id: "kubedl-dashboard-events-dir-required",
                                    }),
                                  },
                                ]}
                                noStyle
                            >
                              <Input
                                  style={{width: "70%"}}
                                  placeholder={"/root/data/log/"}
                              />
                            </Form.Item>
                            <Button
                                type="link"
                                icon={<CheckOutlined/>}
                                onClick={() => onUpdateConfig()}
                            />
                          </Form>
                        </div>
                    ) : (
                        <div>
                          {tbStatus?.config?.logDir}
                          <Button
                              type="link"
                              icon={<EditOutlined/>}
                              onClick={() => onEditConfig()}
                          />
                        </div>
                    )}
                  </Descriptions.Item>
                </Descriptions>
              </div>
          )}
        </div>
    );
  }

  return (
    <Modal
      visible={true}
      title={`${
        isCreating
          ? intl.formatMessage({ id: "kubedl-dashboard-create" })
          : intl.formatMessage({ id: "kubedl-dashboard-view" })
      } Tensorboard`}
      okText={
        isCreating ? intl.formatMessage({ id: "kubedl-dashboard-create" }) : null
      }
      cancelText={intl.formatMessage({ id: "kubedl-dashboard-cancel" })}
      onCancel={onCancel}
      footer={[
        <Button key="back" onClick={onCancel}>
          {intl.formatMessage({ id: "kubedl-dashboard-close" })}
        </Button>,
        isCreating && (
          <Button
            key="submit"
            type="primary"
            onClick={() => {
              createForm.validateFields().then((values) => {
                createForm.resetFields();
                onSubmit(values);
              });
            }}
          >
            {intl.formatMessage({ id: "kubedl-dashboard-create" })}
          </Button>
        ),
      ]}
    >
      {isCreating ? <CreateForm /> : <ViewContainer />}
    </Modal>
  );
};

export default CreateTBModal;

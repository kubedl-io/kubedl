import React, { useState, useRef, useEffect } from "react";
import {
  Button,
  Modal,
  Form,
  Input,
  Descriptions,
  Tooltip,
  Spin,
  Divider,
  message,
} from "antd";
import {
  QuestionCircleTwoTone,
  EditOutlined,
  CheckOutlined,
} from "@ant-design/icons";
import PodStatus from "@/components/PodStatus";
import { getJobTensorboardStatus, reApplyJobTensorboard } from "./service";
import { useIntl } from "umi";

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
          host: window.location.hostname,
          pathPrefix: "/",
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
            host: window.location.hostname,
            pathPrefix: "/",
          },
        }
      );
      message.success(
        intl.formatMessage({ id: "dlc-dashboard-update-log-success" })
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
              intl.formatMessage({ id: "dlc-dashboard-output-log-url" })
            }
          >
            {intl.formatMessage({ id: "dlc-dashboard-events-dir" })}{" "}
            <QuestionCircleTwoTone twoToneColor="#faad14" />
          </Tooltip>
        }
        name={["logDir"]}
        rules={[
          {
            required: true,
            message: intl.formatMessage({
              id: "dlc-dashboard-events-dir-required",
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

  let ViewContainer = () => (
    <div>
      {tbLoading ? (
        <div style={{ textAlign: "center", padding: "30px" }}>
          <Spin />
        </div>
      ) : (
        <div>
          <Descriptions size="small">
            <Descriptions.Item
              label={intl.formatMessage({ id: "dlc-dashboard-domain-name" })}
              span={3}
            >
              {tbStatus.ingresses ? (
                <a
                  target="_blank"
                  href={`${window.location.protocol}//${tbStatus.ingresses[0].spec.rules[0].host}/${selectedJob.namespace}/${selectedJob.name}`}
                >
                  {intl.formatMessage({ id: "dlc-dashboard-open" })} Tensorboard
                </a>
              ) : (
                "-"
              )}
            </Descriptions.Item>
            <Descriptions.Item
              label={intl.formatMessage({ id: "dlc-dashboard-status" })}
              span={3}
            >
              <PodStatus status={tbStatus.phase}></PodStatus>
            </Descriptions.Item>
            <Descriptions.Item
              label={intl.formatMessage({ id: "dlc-dashboard-details" })}
              span={3}
            >
              {tbStatus.message || "-"}
            </Descriptions.Item>
          </Descriptions>
          <Divider />
          <Descriptions size="small">
            <Descriptions.Item
              label={intl.formatMessage({ id: "dlc-dashboard-events-dir" })}
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
                            id: "dlc-dashboard-events-dir-required",
                          }),
                        },
                      ]}
                      noStyle
                    >
                      <Input
                        style={{ width: "70%" }}
                        placeholder={"/root/data/log/"}
                      />
                    </Form.Item>
                    <Button
                      type="link"
                      icon={<CheckOutlined />}
                      onClick={() => onUpdateConfig()}
                    />
                  </Form>
                </div>
              ) : (
                <div>
                  {tbStatus?.config?.logDir}
                  <Button
                    type="link"
                    icon={<EditOutlined />}
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

  return (
    <Modal
      visible={true}
      title={`${
        isCreating
          ? intl.formatMessage({ id: "dlc-dashboard-create" })
          : intl.formatMessage({ id: "dlc-dashboard-view" })
      } Tensorboard`}
      okText={
        isCreating ? intl.formatMessage({ id: "dlc-dashboard-create" }) : null
      }
      cancelText={intl.formatMessage({ id: "dlc-dashboard-cancel" })}
      onCancel={onCancel}
      footer={[
        <Button key="back" onClick={onCancel}>
          {intl.formatMessage({ id: "dlc-dashboard-close" })}
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
            {intl.formatMessage({ id: "dlc-dashboard-create" })}
          </Button>
        ),
      ]}
    >
      {isCreating ? <CreateForm /> : <ViewContainer />}
    </Modal>
  );
};

export default CreateTBModal;

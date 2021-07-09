import {
  ExclamationCircleOutlined,
  FundViewOutlined,
  PlusSquareOutlined,
  FormOutlined,
  DeleteOutlined,
  CopyOutlined,
} from "@ant-design/icons";
import { Modal, message, Tooltip } from "antd";
import React, { useRef, useState, useEffect, Fragment } from "react";
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import ProTable from "@ant-design/pro-table";
import {
  deleteJobs,
  queryJobs,
  stopJobs,
  getJobTensorboardStatus,
} from "./service";
import { cloneInfoJobs } from "../JobDetail/service";
import CreateTBModal from "./CreateTBModal";
import moment from "moment";
import { connect, useIntl, history } from "umi";
import { queryCurrentUser } from "@/services/global";
const TableList = ({ globalConfig }) => {
  const intl = useIntl();
  const [loading, setLoading] = useState(true);
  const [jobs, setJobs] = useState([]);
  const [tbModalVisible, setTbModalVisible] = useState(false);
  const [isViewTensorboard, setIsViewTensorboard] = useState(false);
  const [selectedJob, setSelectedJob] = useState(null);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState({});

  const pageSizeRef = useRef(20);
  const currentRef = useRef(1);
  const paramsRef = useRef({});
  const fetchIntervalRef = useRef();
  const actionRef = useRef();
  const formRef = useRef();

  const searchInitialParameters = {
    jobStatus: "All",
    submitDateRange: [moment().subtract(30, "days"), moment()],
    current: 1,
    page_size: 20,
  };

  useEffect(() => {
    fetchJobs();
    fetchUser();
    const interval = 10 * 1000;
    fetchIntervalRef.current = setInterval(() => {
      fetchJobsSilently();
    }, interval);
    return () => {
      clearInterval(fetchIntervalRef.current);
    };
  }, []);

  const fetchJobs = async () => {
    setLoading(true);
    await fetchJobsSilently();
    setLoading(false);
  };

  const fetchUser = async () => {
    const users = await queryCurrentUser();
    let userInfos = users.data ? users.data : {};
    setUsers(userInfos);
  };

  const fetchJobsSilently = async () => {
    let queryParams = { ...paramsRef.current };
    if (!paramsRef.current.submitDateRange) {
      queryParams = {
        ...queryParams,
        ...searchInitialParameters,
      };
    }
    let jobs = await queryJobs({
      name: queryParams.name,
      // namespace: globalConfig.namespace,
      status:
        queryParams.jobStatus === "All" ? undefined : queryParams.jobStatus,
      start_time: moment(queryParams.submitDateRange[0])
        .hours(0)
        .minutes(0)
        .seconds(0)
        .utc()
        .format(),
      end_time: moment(queryParams.submitDateRange[1])
        .hours(0)
        .minutes(0)
        .seconds(0)
        .add(1, "days")
        .utc()
        .format(),
      current_page: currentRef.current,
      kind: queryParams.jobType,
      page_size: pageSizeRef.current,
    });
    setJobs(jobs.data);
    setTotal(jobs.total);
  };

  const onDetail = (job) => {
    history.push({
      pathname: `/jobs/detail`,
      query: {
        region: job.deployRegion,
        start_date: moment(job.createTime)
          .utc()
          .format("YYYY-MM-DD"),
        job_name: job.name,
        namespace: job.namespace,
        kind: job.jobType,
        current_page: 1,
        page_size: 10,
      },
    });
  };

  const onJobDelete = (job) => {
    Modal.confirm({
      title: intl.formatMessage({ id: "kubedl-dashboard-delete-job" }),
      icon: <ExclamationCircleOutlined />,
      content: `${intl.formatMessage({
        id: "kubedl-dashboard-delete-job-confirm",
      })} ${job.name} ?`,
      onOk: () =>
        deleteJobs(
          job.namespace,
          job.name,
          job.id,
          job.jobType,
          moment(job.submitTime)
            .utc()
            .format()
        ).then(() => {
          const { current } = actionRef;
          if (current) {
            current.reload();
          }
        }),
      onCancel() {},
    });
  };

  const onSearchSubmit = (params) => {
    paramsRef.current = params;
    fetchJobs();
  };

  const onJobStop = (job) => {
    Modal.confirm({
      title: intl.formatMessage({ id: "kubedl-dashboard-termination-job" }),
      icon: <ExclamationCircleOutlined />,
      content: `${intl.formatMessage({
        id: "kubedl-dashboard-termination-job-confirm",
      })} ${job.name} ?`,
      onOk: () => stopJobs(job.deployRegion, job.namespace, job.name),
      onCancel() {},
    });
  };

  const onTBCancel = () => {
    setTbModalVisible(false);
    setSelectedJob(null);
    setIsViewTensorboard(false);
  };

  const onTBCheckAndOpen = async (job) => {
    let res = await getJobTensorboardStatus({
      job_namespace: job.namespace,
      job_name: job.name,
      job_uid: job.id,
      kind: job.jobType,
    });
    let tbStatus = res.data || {};
    if (tbStatus.phase === "Running" && tbStatus.ingresses) {
      let host = tbStatus.ingresses[0].spec.rules[0].host;
      let job_namespace = job.namespace;
      let job_name = job.name;
      window.open(`http://${host}/${job_namespace}/${job_name}`, "_blank");
    } else {
      message.warning(
        "Tensorboard " + intl.formatMessage({ id: "kubedl-dashboard-start-info" }),
        3
      );
    }
  };
  const onTBOpen = (isView, job) => {
    setIsViewTensorboard(isView);
    setSelectedJob(job);
    setTbModalVisible(true);
  };

  const onTableChange = (pagination) => {
    if (pagination) {
      currentRef.current = pagination.current;
      pageSizeRef.current = pagination.pageSize;
      fetchJobs();
    }
  };
  const Tip = ({ kubedl, Click, disabled, IconComponent }) => {
    return (
      <Tooltip title={intl.formatMessage({ id: kubedl })}>
        <a onClick={() => Click()} disabled={disabled}>
          {IconComponent}
        </a>
      </Tooltip>
    );
  };
  const ClickClone = async (record) => {
    try {
      const { data, code } = await cloneInfoJobs(
        record.namespace,
        record.name,
        record.jobType
      );
      if (code == 200) {
        if (JSON.parse(data || "{}").metadata) {
          window.sessionStorage.setItem("job", data);
          history.push({ pathname: "/job-submit" });
        }
      } else {
        message.error(JSON.stringify(data));
      }
    } catch (error) {
      console.log(JSON.stringify(error));
    }
  };
  let columns = [
    {
      // title: 'Date Range',
      title: intl.formatMessage({ id: "kubedl-dashboard-time-interval" }),
      dataIndex: "submitDateRange",
      valueType: "dateRange",
      initialValue: searchInitialParameters.submitDateRange,
      hideInTable: true,
    },
    {
      // title: 'Namespace',
      title: intl.formatMessage({ id: "kubedl-dashboard-namespace" }),
      dataIndex: "namespace",
      hideInSearch: true,
    },
    {
      title: intl.formatMessage({ id: "kubedl-dashboard-job-type" }),
      dataIndex: "jobType",
      valueEnum: {
        PyTorchJob: {
          text: "PyTorchJob",
          status: "Default",
        },
        TFJob: {
          text: "TFJob",
          status: "Default",
        },
        MPIJob: {
          text: "MPIJob",
          status: "Default",
        },
      },
    },
    {
      // title: 'Status',
      title: intl.formatMessage({ id: "kubedl-dashboard-status" }),
      width: 128,
      dataIndex: "jobStatus",
      initialValue: searchInitialParameters.jobStatus,
      valueEnum: {
        All: {
          text: intl.formatMessage({ id: "kubedl-dashboard-all" }),
          // text: 'All',
          status: "Default",
        },
        Created: {
          text: intl.formatMessage({ id: "kubedl-dashboard-has-created" }),
          // text: 'Created',
          status: "Default",
        },
        Waiting: {
          text: intl.formatMessage({ id: "kubedl-dashboard-waiting-for" }),
          // text: 'Waiting',
          status: "Processing",
        },
        Running: {
          text: intl.formatMessage({ id: "kubedl-dashboard-executing" }),
          // text: 'Running',
          status: "Processing",
        },
        Succeeded: {
          text: intl.formatMessage({ id: "kubedl-dashboard-execute-success" }),
          // text: 'Succeeded',
          status: "Success",
        },
        Failed: {
          text: intl.formatMessage({ id: "kubedl-dashboard-execute-failure" }),
          // text: 'Failed',
          status: "Error",
        },
        Stopped: {
          text: intl.formatMessage({ id: "kubedl-dashboard-has-stopped" }),
          // text: 'Stopped',
          status: "Error",
        },
      },
    },
    {
      // title: 'Create Time',
      title: intl.formatMessage({ id: "kubedl-dashboard-creation-time" }),
      dataIndex: "createTime",
      //valueType: "date",
      hideInSearch: true,
    },
    {
      // title: 'End Time',
      title: intl.formatMessage({ id: "kubedl-dashboard-end-time" }),
      dataIndex: "endTime",
      //valueType: "date",
      hideInSearch: true,
    },
    {
      width: 142,
      title: intl.formatMessage({ id: "kubedl-dashboard-execution-time" }),
      dataIndex: "durationTime",
      hideInSearch: true,
      render: (text) => <Fragment>{text && text.split(".")[0]}</Fragment>,
    },
    {
      title: intl.formatMessage({ id: "kubedl-dashboard-operation" }),
      dataIndex: "option",
      valueType: "option",
      render: (_, record) => {
        let isDisabled;
        if (users.loginId && users.loginId !== "") {
          isDisabled = true;
        } 
        let iconStyleMarginLeft = {
          marginLeft: "10px",
          color: isDisabled ? "#1890ff" : "",
        };
        return (
          <Fragment>
            <Tip
              kubedl={"kubedl-dashboard-clone"}
              Click={ClickClone.bind(this, record)}
              disabled={!isDisabled}
              IconComponent={
                <CopyOutlined
                  style={{
                    marginRight: "10px",
                    color: isDisabled ? "#1890ff" : "",
                  }}
                />
              }
            />
            <Tip
              kubedl={"kubedl-dashboard-delete"}
              Click={onJobDelete.bind(this, record)}
              disabled={!isDisabled}
              IconComponent={
                <DeleteOutlined
                  style={{ color: isDisabled ? "#d9363e" : "" }}
                />
              }
            />
            {environment && (
              <>
                {record.enableTensorboard === true ? (
                  <>
                    <Tip
                      kubedl={"kubedl-dashboard-open"}
                      Click={onTBCheckAndOpen.bind(this, record)}
                      disabled={!isDisabled}
                      IconComponent={
                        <FundViewOutlined style={iconStyleMarginLeft} />
                      }
                    />
                    <Tip
                      kubedl={"kubedl-dashboard-edit"}
                      Click={onTBOpen.bind(this, true, record)}
                      disabled={!isDisabled}
                      IconComponent={
                        <FormOutlined style={iconStyleMarginLeft} />
                      }
                    />
                  </>
                ) : record.enableTensorboard === false ? (
                  <Tip
                    kubedl={"kubedl-dashboard-create"}
                    Click={onTBOpen.bind(this, false, record)}
                    disabled={!isDisabled}
                    IconComponent={
                      <PlusSquareOutlined style={iconStyleMarginLeft} />
                    }
                  />
                ) : (
                  ""
                )}
              </>
            )}
          </Fragment>
        );
      },
    },
  ];
  let nameAndUser = [
    {
      title: intl.formatMessage({ id: "kubedl-dashboard-name" }),
      dataIndex: "name",
      width: 196,
      render: (_, r) => {
        return <a onClick={() => onDetail(r)}>{r.name}</a>;
      },
    },
  ];
  if (environment) {
    nameAndUser = [
      ...nameAndUser,
      {
        title: intl.formatMessage({ id: "kubedl-dashboard-user" }),
        dataIndex: "jobUserName",
        hideInSearch: true,
        render: (_, r) => {
          const name =
            r.jobUserName && r.jobUserName !== "" ? r.jobUserName : r.jobUserId;
          return <span>{name}</span>;
        },
      },
    ];
  }
  return (
    <div>
      <ProTable
        loading={loading}
        dataSource={jobs}
        onSubmit={(params) => onSearchSubmit(params)}
        headerTitle={intl.formatMessage({ id: "kubedl-dashboard-job-list" })}
        actionRef={actionRef}
        formRef={formRef}
        rowKey="key"
        columns={[...nameAndUser, ...columns]}
        options={{
          fullScreen: true,
          setting: true,
          reload: () => fetchJobs(),
        }}
        onChange={onTableChange}
        pagination={{ total: total }}
        scroll={{ y: 450 }}
      />
      {tbModalVisible && (
        <CreateTBModal
          selectedJob={selectedJob}
          isViewing={isViewTensorboard}
          onCancel={() => onTBCancel()}
        />
      )}
    </div>
  );
};

export default connect(({ global }) => ({
  globalConfig: global.config,
}))(TableList);

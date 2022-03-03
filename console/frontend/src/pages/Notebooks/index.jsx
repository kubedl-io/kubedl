import {CopyOutlined, DeleteOutlined, ExclamationCircleOutlined, FundViewOutlined} from "@ant-design/icons";
import {message, Modal, Tooltip} from "antd";
import React, {Fragment, useEffect, useRef, useState} from "react";
import ProTable from "@ant-design/pro-table";
import {cloneInfoNotebook, deleteNotebook, queryNotebooks,} from "./service";
import moment from "moment";
import {connect, history, useIntl} from "umi";
import {queryCurrentUser} from "@/services/global";

const TableList = ({ globalConfig, namespace }) => {
  const intl = useIntl();
  const [loading, setLoading] = useState(true);
  const [notebooks, setNotebooks] = useState([]);
  const [tbModalVisible, setTbModalVisible] = useState(false);
  const [selectedNotebook, setSelectedNotebook] = useState(null);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState({});

  const pageSizeRef = useRef(20);
  const currentRef = useRef(1);
  const paramsRef = useRef({});
  const fetchIntervalRef = useRef();
  const actionRef = useRef();
  const formRef = useRef();

  const searchInitialParameters = {
    notebookStatus: "All",
    submitDateRange: [moment().subtract(30, "days"), moment()],
    current: 1,
    page_size: 20,
    namespace: namespace
  };

  useEffect(() => {
    fetchNotebooks();
    fetchUser();
    const interval = 10 * 1000;
    fetchIntervalRef.current = setInterval(() => {
      fetchNotebooksSilently();
    }, interval);
    return () => {
      clearInterval(fetchIntervalRef.current);
    };
  }, []);

  const fetchNotebooks = async () => {
    setLoading(true);
    await fetchNotebooksSilently();
    setLoading(false);
  };

  const fetchUser = async () => {
    const users = await queryCurrentUser();
    let userInfos = users.data ? users.data : {};
    setUsers(userInfos);
  };

  const fetchNotebooksSilently = async () => {
    let queryParams = { ...paramsRef.current };
    if (!paramsRef.current.submitDateRange) {
      queryParams = {
        ...queryParams,
        ...searchInitialParameters,
      };
    }
    let notebooks = await queryNotebooks({
      name: queryParams.name,
      // namespace: globalConfig.namespace,
      namespace: queryParams.namespace,
      status:
          queryParams.notebookStatus === "All" ? undefined : queryParams.notebookStatus,
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
      page_size: pageSizeRef.current,
    });

    notebooks.data.sort((a, b) => {
      var dict = {
        "Created": 0,
        "Running": 1,
        "Terminated": -1
      };
      let aStatus = dict[a.notebookStatus];
      let bStatus = dict[b.notebookStatus];
      if (aStatus !== bStatus) {
        return aStatus - bStatus
      } else {
        return a.name.localeCompare(b.name)
      }
    });
    setNotebooks(notebooks.data);
    setTotal(notebooks.total);
  };

  const onDetail = (notebook) => {
    history.push({
      pathname: `/notebooks/detail`, //TODO NOT IMPLEMENTED
      query: {
        region: notebook.deployRegion,
        start_date: moment(notebook.createTime)
            .utc()
            .format("YYYY-MM-DD"),
        job_name: notebook.name,
        namespace: notebook.namespace,
        current_page: 1,
        page_size: 10,
      },
    });
  };

  const onNotebookUrl = (notebook) => {
    // window.location = notebook.url
    if (notebook.url !== '') {
      window.open(notebook.url, '_blank');
    } else {
      message.warning(
          "Notebook " + intl.formatMessage({id: "kubedl-dashboard-start-info"}),
          3
      );
    }
  };

  const onDelete = (notebook) => {
    Modal.confirm({
      title: intl.formatMessage({ id: "kubedl-dashboard-delete-notebook" }),
      icon: <ExclamationCircleOutlined />,
      content: `${intl.formatMessage({
        id: "kubedl-dashboard-delete-notebook-confirm",
      })} ${notebook.name} ?`,
      onOk: () =>
          deleteNotebook(
              notebook.namespace,
              notebook.name,
              notebook.id,
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
    paramsRef.current = params; //TODO WHAT IS paramRef
    fetchNotebooks();
  };

  const onJobStop = (job) => { //TODO
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

  const onTableChange = (pagination) => {
    if (pagination) {
      currentRef.current = pagination.current;
      pageSizeRef.current = pagination.pageSize;
      fetchNotebooks();
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
  const ClickClone = async (record) => { //TODO
    try {
      const { data, code } = await cloneInfoNotebook(
          record.namespace,
          record.name,
      );
      if (code === "200") {
        if (JSON.parse(data || "{}").metadata) {
          window.sessionStorage.setItem("notebook", data);
          history.push({ pathname: "/notebook-submit" });
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
      hideInSearch: true,
    },
    {
      // title: 'Namespace',
      title: intl.formatMessage({ id: "kubedl-dashboard-namespace" }),
      dataIndex: "namespace",
    },
    {
      // title: 'Status',
      title: intl.formatMessage({ id: "kubedl-dashboard-status" }),
      width: 128,
      dataIndex: "notebookStatus",
      initialValue: searchInitialParameters.notebookStatus,
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
        Running: {
          text: intl.formatMessage({ id: "kubedl-dashboard-executing" }),
          // text: 'Running',
          status: "Processing",
        },
        Terminated: {
          text: intl.formatMessage({ id: "kubedl-dashboard-execute-terminated" }),
          // text: 'Terminated',
          status: "Success",
        }
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
      render: (text) => <Fragment>{text}</Fragment>,
    },
    {
      width: 142,
      title: intl.formatMessage({ id: "kubedl-dashboard-notebook-url" }),
      dataIndex: "url",
      hideInSearch: true,
      render: (_, record) => {
        let isDisabled;
        if (users.loginId && users.loginId !== "") {
          isDisabled = true;
        }
        if (record.url === "") {
          isDisabled = true;
        }
        return (
            <Fragment>
              <Tip
                  kubedl={'kubedl-dashboard-open'}
                  Click={onNotebookUrl.bind(this, record)}
                  disabled={!isDisabled}
                  IconComponent={
                    <FundViewOutlined
                        style={{
                          marginRight: "10px",
                          color: isDisabled ? "#1890ff" : "",
                        }}
                    />
                  }
              />
            </Fragment>
        );
      },
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
                  Click={onDelete.bind(this, record)}
                  disabled={!isDisabled}
                  IconComponent={
                    <DeleteOutlined
                        style={{ color: isDisabled ? "#d9363e" : "" }}
                    />
                  }
              />
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
      // render: (_, r) => {
      //   return <a onClick={() => onDetail(r)}>{r.name}</a>;
      // },
    },
  ];
  if (environment) {
    nameAndUser = [
      ...nameAndUser,
      {
        title: intl.formatMessage({ id: "kubedl-dashboard-user" }),
        dataIndex: "userName",
        hideInSearch: true,
        render: (_, r) => {
          const name =
              r.userName && r.userName !== "" ? r.userName : "Anonymous";
          return <span>{name}</span>;
        },
      },
    ];
  }
  return (
      <div>
        <ProTable
            loading={loading}
            dataSource={notebooks}
            onSubmit={(params) => onSearchSubmit(params)}
            headerTitle={intl.formatMessage({ id: "kubedl-dashboard-notebook-list" })}
            actionRef={actionRef}
            formRef={formRef}
            rowKey="key"
            search={{
              filterType: 'light',
            }}
            columns={[...nameAndUser, ...columns]}
            options={{
              fullScreen: true,
              setting: true,
              reload: () => fetchNotebooks(),
            }}
            onChange={onTableChange}
            pagination={{ total: total }}
            scroll={{ y: 450 }}
        />
      </div>
  );
};

export default connect(({ global }) => ({
  globalConfig: global.config,
}))(TableList);

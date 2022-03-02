import {DeleteOutlined, ExclamationCircleOutlined,} from "@ant-design/icons";
import {Button, Card, Modal, Tooltip} from "antd";
import React, {Fragment, useEffect, useRef, useState} from "react";
import ProTable from "@ant-design/pro-table";
import {deleteWorkspace, queryWorkspaces,} from "./service";
import moment from "moment";
import {connect, history, useIntl} from "umi";
import {queryCurrentUser} from "@/services/global";

const TableList = ({ globalConfig }) => {
  const intl = useIntl();
  const [loading, setLoading] = useState(true);
  const [workspaces, setWorkspaces] = useState([]);
  const [total, setTotal] = useState(0);
  const [users, setUsers] = useState({});

  const pageSizeRef = useRef(20);
  const currentRef = useRef(1);
  const paramsRef = useRef({});
  const fetchIntervalRef = useRef();
  const actionRef = useRef();
  const formRef = useRef();

  const searchInitialParameters = {
    submitDateRange: [moment().subtract(30, "years"), moment()],
    current: 1,
    page_size: 20,
  };

  useEffect(() => {
    fetchWorkspaces();
    fetchUser();
    const interval = 10 * 1000;
    fetchIntervalRef.current = setInterval(() => {
      fetchWorkspacesSilently();
    }, interval);
    return () => {
      clearInterval(fetchIntervalRef.current);
    };
  }, []);

  const fetchWorkspaces = async () => {
    setLoading(true);
    await fetchWorkspacesSilently();
    setLoading(false);
  };

  const fetchUser = async () => {
    const users = await queryCurrentUser();
    let userInfos = users.data ? users.data : {};
    setUsers(userInfos);
  };

  const fetchWorkspacesSilently = async () => {
    let queryParams = { ...paramsRef.current };
    if (!paramsRef.current.submitDateRange) {
      queryParams = {
        ...queryParams,
        ...searchInitialParameters,
      };
    }
    let workspaces = await queryWorkspaces({
      name: queryParams.name,
      // namespace: globalConfig.namespace,
      // status:
      //     queryParams.notebookStatus === "All" ? undefined : queryParams.notebookStatus,
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
    workspaces.data.sort((a, b) => a.name.localeCompare(b.name));
    setWorkspaces(workspaces.data);
    console.log("workspaces: " + JSON.stringify(workspaces.data))
    setTotal(workspaces.total);
  };

  const onDetail = (workspace) => {
    history.push({
      pathname: `/workspace/detail`, //TODO NOT IMPLEMENTED
      query: {
        name: workspace.name,
        current_page: 1,
        page_size: 10,
      },
    });
  };

  const onDelete = (workspace) => {
    Modal.confirm({
      title: intl.formatMessage({ id: "kubedl-dashboard-delete-workspace" }),
      icon: <ExclamationCircleOutlined />,
      content: `${intl.formatMessage({
        id: "kubedl-dashboard-delete-workspace-confirm",
      })} ${workspace.name} ?`,
      onOk: () =>
          deleteWorkspace(
              workspace.name,
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
    fetchWorkspaces();
  };


  const onTableChange = (pagination) => {
    if (pagination) {
      currentRef.current = pagination.current;
      pageSizeRef.current = pagination.pageSize;
      fetchWorkspaces();
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

  let columns = [
    // {
    //   // title: 'Date Range',
    //   title: intl.formatMessage({ id: "kubedl-dashboard-time-interval" }),
    //   dataIndex: "submitDateRange",
    //   valueType: "dateRange",
    //   initialValue: searchInitialParameters.submitDateRange,
    //   hideInTable: true,
    // },
    // {
    //   // title: 'Namespace',
    //   title: intl.formatMessage({ id: "kubedl-dashboard-namespace" }),
    //   dataIndex: "namespace",
    //   hideInSearch: true,
    // },
    {
      // title: 'Create Time',
      title: intl.formatMessage({ id: "kubedl-dashboard-creation-time" }),
      dataIndex: "create_time",
      //valueType: "date",
      hideInSearch: true,
    },
    {
      // title: 'Status',
      title: intl.formatMessage({ id: "kubedl-dashboard-status" }),
      width: 128,
      dataIndex: "status",
      valueEnum: {
        Created: {
          text: intl.formatMessage({ id: "kubedl-dashboard-has-created" }),
          // text: 'Created',
          status: "Default",
        },
        Ready: {
          text: intl.formatMessage({ id: "kubedl-dashboard-executing" }),
          // text: 'Running',
          status: "Processing",
        },
        Waiting: {
          text: intl.formatMessage({ id: "kubedl-dashboard-workspace-waiting" }),
          // text: 'StorageLost',
          status: "Error",
        },
      },
    },
    {
      width: 142,
      title: intl.formatMessage({ id: "kubedl-dashboard-execution-time" }),
      dataIndex: "duration_time",
      hideInSearch: true,
      render: (text) => <Fragment>{text}</Fragment>,
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
              {/*<Tip*/}
              {/*    kubedl={"kubedl-dashboard-clone"}*/}
              {/*    Click={ClickClone.bind(this, record)}*/}
              {/*    disabled={!isDisabled}*/}
              {/*    IconComponent={*/}
              {/*      <CopyOutlined*/}
              {/*          style={{*/}
              {/*            marginRight: "10px",*/}
              {/*            color: isDisabled ? "#1890ff" : "",*/}
              {/*          }}*/}
              {/*      />*/}
              {/*    }*/}
              {/*/>*/}
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
      hideInSearch: true,
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

  const onWorkspaceCreate = () => {
    history.push({
      pathname: `/workspace-create`,
      query: {}
    });
  };

  return (
      <div>
        <Card style={{ marginBottom: 12 }}
              //TODO disable new workspace for now
        //       title={
        //   <div style={{paddingBottom:"8px", textAlign: "right",display: 'inline-block', marginRight: '20px'}}>
        //     <Button type="primary" onClick={onWorkspaceCreate} style={{minWidth: '120px'}} size='small'>
        //       {intl.formatMessage({id: 'kubedl-dashboard-new-workspace'})}
        //     </Button>
        //   </div>
        // }
        >
          <ProTable
              loading={loading}
              dataSource={workspaces}
              search={false}
              onSubmit={(params) => onSearchSubmit(params)}
              headerTitle={intl.formatMessage({id: "kubedl-dashboard-workspace-list"})}
              actionRef={actionRef}
              formRef={formRef}
              rowKey="key"
              columns={[...nameAndUser, ...columns]}
              options={{
                fullScreen: true,
                setting: true,
                reload: () => fetchWorkspaces(),
              }}
              onChange={onTableChange}
              pagination={{total: total}}
              scroll={{y: 450}}
          />
        </Card>
      </div>
  );
};

export default connect(({ global }) => ({
  globalConfig: global.config,
}))(TableList);

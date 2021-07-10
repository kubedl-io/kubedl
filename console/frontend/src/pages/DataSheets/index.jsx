import { Button, Tabs, Card, Modal, Tooltip, message} from "antd";
import React, {useState, useEffect, useRef} from "react";
import { history, useIntl } from 'umi';
import { PageHeaderWrapper } from "@ant-design/pro-layout";
import ProTable from "@ant-design/pro-table";
import * as service  from "./service";
import { CodeOutlined, CloudServerOutlined, ExclamationCircleOutlined, DeleteOutlined} from '@ant-design/icons';
import styles from './index.less';

const { TabPane } = Tabs;

const DataSheetsList = () => {
  const intl = useIntl();
  const [loading, setLoading] = useState(true);
  const [datasource, setDatasource] = useState([]);
  const [codesource, setCodesource] = useState([]);
  const pageSizeRef = useRef(20);
  const currentRef = useRef(1);

  useEffect(() => {
    fetchInit();
  }, []);
  const fetchInit =()=>{
    fetchDataRequest();
    fetchCodeRequest();
  }
  const fetchDataRequest = async () => {
    setLoading(true);
    let dataRequest = await service.getDatasources();
    const datasource = [];
    if (dataRequest.code !== "300" && dataRequest.data) {
      for (const key in dataRequest.data) {
        datasource.push(dataRequest.data[key]);
      }
    }
    setDatasource(datasource);
    setLoading(false);
  }

  const fetchCodeRequest = async () => {
    setLoading(true);
    let codeRequest = await service.getCodesources();
    const codesource = [];
    if (codeRequest.code !== "300" && codeRequest.data) {
      for (const key in codeRequest.data) {
        codesource.push(codeRequest.data[key]);
      }
    }
    setCodesource(codesource);
    setLoading(false);
  }
  const onDelete = ({titleIntl, content, reqInterface, params}) => {
    Modal.confirm({
      title: intl.formatMessage({id: titleIntl}),
      icon: <ExclamationCircleOutlined />,
      content: `${content || ""}`,
      onOk: async()=>{
        try{
          let {data, code} = await service[reqInterface](params);
          if(code !=200){
              message.error(JSON.stringify(data))
          }else {
            fetchInit();
          }
        }catch(error){
           console.log(JSON.stringify(error));
        }
        
      },
    });
  };
  let columnsData = [
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-name'}),
      dataIndex: "name",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-pvc-name'}),
      dataIndex: "pvc_name",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-namespace'}),
      dataIndex: "namespace",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-local-paths'}),
      dataIndex: "local_path",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-creator'}),
      dataIndex: "user",
      render: (_, record) => {
       const user = record && record.username !== '' ? record.username : (record &&record.userid || "");
        return (
            <>
              {user}
            </>
        )
      }
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-description'}),
      dataIndex: "description",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-creation-time'}),
      dataIndex: "create_time",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-update-time'}),
      dataIndex: "update_time",
    }
  ];
  if(environment){
    columnsData =[
      ...columnsData,
      {
        title: intl.formatMessage({id: 'kubedl-dashboard-operation'}),
        dataIndex: "option",
        valueType: "option",
        render: (_, record) => {
          let {name} = record || {};
          return (
            <>
              <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-delete'})}>
                <a onClick={
                  () => onDelete({titleIntl:"kubedl-dashboard-delete-data-configuration",
                                  reqInterface:"deleteDataConfig",
                                  params:{name},
                                  content:`${intl.formatMessage({id: 'kubedl-dashboard-confirm-to-delete-data-configuration'})} ${name} ?`,
                                })
                }>
                  <DeleteOutlined style={{color:"red"}}/>
                </a>
              </Tooltip>
            </>
          )
        }
      }
    ]
  };
  let columnsGit = [
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-name'}),
      dataIndex: "name",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-git-repository'}),
      dataIndex: "code_path",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-default-branch'}),
      dataIndex: "default_branch",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-local-paths'}),
      dataIndex: "local_path",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-creator'}),
      dataIndex: "user",
      render: (_, record) => {
        const user = record.username && record.username !== '' ? record.username : record.userid;
        return (
            <>
              {user}
            </>
        )
      }
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-description'}),
      dataIndex: "description",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-creation-time'}),
      dataIndex: "create_time",
    },
    {
      title: intl.formatMessage({id: 'kubedl-dashboard-update-time'}),
      dataIndex: "update_time",
    }
  ];
  if(environment){
    columnsGit =[
      ...columnsGit,
      {
        title: intl.formatMessage({id: 'kubedl-dashboard-operation'}),
        dataIndex: "option",
        valueType: "option",
        render: (_, record) => {
          let {name} = record;
          return (
            <>
              <Tooltip title={intl.formatMessage({id: 'kubedl-dashboard-delete'})}>
                <a onClick={
                  () => onDelete({titleIntl:"kubedl-dashboard-delete-code-configuration",
                                  reqInterface:"deleteGitConfig",
                                  params:{name},
                                  content:`${intl.formatMessage({id: 'kubedl-dashboard-confirm-to-delete-code-configuration'})} ${name} ?`,
                                })
                }>
                  <DeleteOutlined style={{color:"red"}}/>
                </a>
              </Tooltip>
            </>
          )
        }
      }
    ]
  }
  const onDataConfigCreate = () => {
    history.push({
      pathname: `/datasheets/data-config`,
      query: {}
    });
  };
  const onCodeConfigCreate = () => {
    history.push({
      pathname: `/datasheets/git-config`,
      query: {}
    });
  };
  const onDataSourceTableChange = (pagination) => {
    if (pagination) {
      currentRef.current = pagination.current;
      pageSizeRef.current = pagination.pageSize;
      fetchDataRequest();
    }
  }
  const onCodeSourceTableChange = (pagination) => {
    if (pagination) {
      currentRef.current = pagination.current;
      pageSizeRef.current = pagination.pageSize;
      fetchCodeRequest();
    }
  }
  return (
    <div>
      <Card style={{ marginBottom: 12 }} title={
        <div>
          <div style={{paddingBottom:"8px", textAlign: "right",display: 'inline-block', marginRight: '20px'}}>
            <Button type="primary" onClick={onDataConfigCreate} style={{minWidth: '120px'}} size='small'>
              {intl.formatMessage({id: 'kubedl-dashboard-new-data-config'})}
            </Button>
          </div>
          <div style={{paddingBottom:"8px", textAlign: "right",display: 'inline-block'}} >
            <Button type="primary" onClick={onCodeConfigCreate} style={{minWidth: '120px'}} size='small'>
              {intl.formatMessage({id: 'kubedl-dashboard-new-create-git-config'})}
            </Button>
          </div>
        </div>
      }>
        <Tabs defaultActiveKey="1" type="card">
          <TabPane tab={
            <span>
                <CloudServerOutlined />
                {intl.formatMessage({id: 'kubedl-dashboard-data'})}
              </span>
          } key="data">
            <ProTable
                loading={loading}
                dataSource={datasource}
                toolBarRender={false}
                rowKey="key"
                columns={columnsData}
                pagination={true}
                search={false}
                onChange={onDataSourceTableChange}
            />
          </TabPane>
          <TabPane tab={
            <span>
              <CodeOutlined />
              {intl.formatMessage({id: 'kubedl-dashboard-code'})}
            </span>
          } key="code">
            <ProTable
                loading={loading}
                dataSource={codesource}
                toolBarRender={false}
                rowKey="key"
                columns={columnsGit}
                pagination={true}
                search={false}
                onChange={onCodeSourceTableChange}
            />
          </TabPane>
        </Tabs>
      </Card>
    </div>
  );
};

export default DataSheetsList;

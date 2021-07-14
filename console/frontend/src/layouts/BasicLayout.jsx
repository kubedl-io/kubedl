/**
 * Ant Design Pro v4 use `@ant-design/pro-layout` to handle Layout.
 * You can view component api by:
 * https://github.com/ant-design/ant-design-pro-layout
 */
import ProLayout, { DefaultFooter } from "@ant-design/pro-layout";
import React, { useEffect,useState } from "react";
import { Link,getLocale,setLocale,formatMessage,connect,useIntl } from "umi";
import {
  GithubOutlined,
  MenuUnfoldOutlined,
  MenuFoldOutlined
} from "@ant-design/icons";
import { Result, Button, Select,Menu,Dropdown } from "antd";
import Authorized from "@/utils/Authorized";
import RightContent from "@/components/GlobalHeader/RightContent";
import AvatarDropdown from '@/components/GlobalHeader/AvatarDropdown'
import { isAntDesignPro, getAuthorityFromRouter } from "@/utils/utils";
import { PageLoading } from "@ant-design/pro-layout";
import logo from "../assets/logo.svg";
import "./index.less";
const noMatch = (
  <Result
    status="403"
    title="403"
    subTitle="Sorry, you are not authorized to access this page."
    extra={
      <Button type="primary">
        <Link to="/user/login">Go Login</Link>
      </Button>
    }
  />
);
const { Option } = Select;

/**
 * use Authorized check all menu item
 */
const menuDataRender = menuList =>
  menuList.map(item => {
    const localItem = {
      ...item,
      children: item.children ? menuDataRender(item.children) : []
    };
    return Authorized.check(item.authority, localItem, null);
  });

const BasicLayout = props => {
  const {
    dispatch,
    children,
    settings,
    collapsed,
    config,
    configLoading,
    location = {
      pathname: "/"
    }
  } = props;
 
  const [namespaceValue, setNamespaceValue] = useState('');
  useEffect(() => {
    if (dispatch) {
      dispatch({
        type: "global/fetchNamespaces"
      });
      dispatch({
        type: "global/fetchConfig"
      });
      handleMenuCollapse(true)
    }
  }, []);

  useEffect(() => {
    if(sessionStorage.getItem('namespace')){
      setNamespaceValue(sessionStorage.getItem('namespace'));
    }else {
      if(config){
        setNamespaceValue('default');
      }
    }
  }, [config]);
  /**
   * init variables
   */

  const handleMenuCollapse = payload => {
      if (dispatch) {
          dispatch({
              type: "global/changeLayoutCollapsed",
              payload: !props.collapsed
          });
      }
  }; // get children authority

  const onChangeNamespace = v => {
    sessionStorage.setItem("namespace", v);
    // setNamespaceValue(sessionStorage.getItem('namespace'));
    window.location.reload();
  };

  const authorized = getAuthorityFromRouter(
    props.route.routes,
    location.pathname || "/"
  ) || {
    authority: undefined
  };

  if (configLoading || !config) {
    return <PageLoading />;
  }
   const changLang = () => {
    const locale = getLocale();
    if (!locale || locale === 'zh-CN') {
        setLocale('en-US',true);
    } else {
        setLocale('zh-CN',true);
    }
  };
  return (
    <ProLayout
      logo={logo}
      breakpoint="xxl"
      formatMessage={formatMessage}
      title={<div style={{position:"relative"}}>KubeDL <br/></div>}
      menuHeaderRender={(logoDom, titleDom) => (
        <Link to="/">
          {logoDom}
          {titleDom}
        </Link>
      )}
      onCollapse={handleMenuCollapse}
      menuItemRender={(menuItemProps, defaultDom) => {
        if (
          menuItemProps.isUrl ||
          menuItemProps.children ||
          !menuItemProps.path
        ) {
          return defaultDom;
        }

        return <Link to={menuItemProps.path}>{defaultDom}</Link>;
      }}

      itemRender={(route, params, routes, paths) => {
          return <Link to={route.path}>{route.breadcrumbName}</Link>;
      }}
      // footerRender={footerRender}
      menuDataRender={menuDataRender}
      rightContentRender={() => <RightContent />}
      headerRender={() => (
        <>
            <div style={{
              position: 'absolute',
              right: '30px'}}>
                <span style={{marginRight: '10px'}}>
                    <AvatarDropdown />
                </span>
                <Select defaultValue={getLocale() === 'zh-CN' ? 'chinese' : 'english'} style={{ width: 120 }} onChange={changLang}>
                    <Option value="chinese">简体中文</Option>
                    <Option value="english">English</Option>
                </Select>
            </div>
          {React.createElement(
            props.collapsed ? MenuUnfoldOutlined : MenuFoldOutlined,
            {
              className: "trigger",
              onClick: handleMenuCollapse
            }
          )}
          {/*<Select*/}
          {/*  style={{ width: 225 }}*/}
          {/*  value={namespaceValue}*/}
          {/*  placeholder="请选择集群ID"*/}
          {/*  onChange={onChangeNamespace}*/}
          {/*>*/}
          {/*  {props.namespaces.length > 0 && props.namespaces.map((item) => <Option key={item.value} value={item.value}>{item.label}</Option>)}*/}
          {/*</Select>*/}
        </>
      )}
      {...props}
      {...settings}
    >
      <Authorized authority={authorized.authority} noMatch={noMatch}>
        {children}
      </Authorized>
    </ProLayout>
  );
};
export default connect(({ global, settings, loading }) => ({
  collapsed: false,
  config: global.config,
  namespaces: global.namespaces,
  configLoading: loading.effects["global/fetchConfig"],
  settings
}))(BasicLayout);

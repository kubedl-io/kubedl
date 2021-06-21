import {
  LogoutOutlined,
  SettingOutlined,
  UserOutlined
} from "@ant-design/icons";
import { Avatar, Menu, Spin } from "antd";
import React from "react";
import { history,connect,formatMessage } from "umi";
import HeaderDropdown from "../HeaderDropdown";
import styles from "./index.less";

const notLoggedIn = formatMessage({
  id: 'dlc-dashboard-not-logged-in'
});

class AvatarDropdown extends React.Component {
  onMenuClick = event => {
    const key = event.key;
    const dispatch = this.props.dispatch;
    if(environment && environment !=="eflops"){
      if (key === "logout") {
        if (dispatch) {
          dispatch({
            type: "login/logout"
          });
        }
        return;
      }
      history.push(`/account/${key}`);
    }else {
      dispatch({
        type:"user/fetchLoginOut",
        payload:{}
      });
      window.setTimeout(()=>{
        history.push('/login');
      },1500);
    }
  };

  render() {
    const {
      currentUser = {
        avatar: "",
        name: ""
      },
      menu,
      userLogin
    } = this.props;
    const menuHeaderDropdown = (
      <Menu
        className={styles.menu}
        selectedKeys={[]}
        onClick={this.onMenuClick}
      >
        {menu && (
          <Menu.Item key="center">
            <UserOutlined />
            个人中心
          </Menu.Item>
        )}
        {menu && (
          <Menu.Item key="settings">
            <SettingOutlined />
            个人设置
          </Menu.Item>
        )}
        {menu && <Menu.Divider />}

        <Menu.Item key="logout">
          <LogoutOutlined />
           退出登录
         </Menu.Item>
      </Menu>
    );
    return currentUser && currentUser.loginName ? (
      <HeaderDropdown overlay={menuHeaderDropdown}>
        <span className={`${styles.action} ${styles.account}`}>
          <Avatar
            size="small"
            icon={<UserOutlined />}
            style={{ color: "#1890ff", marginRight: 12 }}
          />
          <span className={styles.name}>{currentUser.loginName}</span>
        </span>
      </HeaderDropdown>
    ) : (
      // <Spin
      //   size="small"
      //   style={{
      //     marginLeft: 8,
      //     marginRight: 8
      //   }}
      // />
    <span className={styles.name}>{notLoggedIn}</span>
    );
  }
}

export default connect(({ user }) => ({
  currentUser: user.currentUser,
  userLogin: user.userLogin
}))(AvatarDropdown);

import {
  LogoutOutlined,
  SettingOutlined,
  UserDeleteOutlined,
  UserOutlined
} from "@ant-design/icons";
import { Avatar, Menu, Spin } from "antd";
import React, {useEffect}  from "react";
import { history,connect,formatMessage } from "umi";
import HeaderDropdown from "../HeaderDropdown";
import styles from "./index.less";

const notLoggedIn = formatMessage({
  id: 'kubedl-dashboard-not-logged-in'
});

const AvatarDropdown = props => {
  const {
    dispatch,
    currentUser,
  } = props;

  const onMenuClick = event => {
    const key = event.key;
    if (key === "logout") {
      dispatch({
        type: "user/fetchLoginOut"
      });
      props.history.push("/");
    }
  };

    const menuHeaderDropdown = (
      <Menu
        className={styles.menu}
        selectedKeys={[]}
        onClick={onMenuClick}
      >
        <Menu.Item key="logout">
          <LogoutOutlined />
           退出登录
         </Menu.Item>
      </Menu>
    );

    return currentUser && currentUser.loginId ? (
      <HeaderDropdown overlay={menuHeaderDropdown}>
        <span className={`${styles.action} ${styles.account}`}>
          <Avatar
            size="small"
            icon={<UserOutlined />}
            style={{ color: "#1890ff", marginRight: 12 }}
          />
          <span className={styles.name}>{currentUser.loginId}</span>
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

export default connect(({ user }) => ({
  currentUser: user.currentUser,
  userLogin: user.userLogin
}))(AvatarDropdown);

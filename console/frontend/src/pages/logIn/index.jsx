import React from "react";
import { Form, Input, Button } from "antd";
import { UserOutlined, KeyOutlined } from "@ant-design/icons";
import styles from "./index.less";
import 'antd/dist/antd.css';
import { connect} from "umi";
const Login =({dispatch})=> {
   const onFinish=({ username, password })=> {
       if (dispatch) {
         dispatch({
           type: "user/fetchLogin",
           payload: { username, password },
         });
       };
  }
    return (
      <div className={styles.loginBox}>
        <div className={styles.loginBoxWindow}>
          <div className={styles.loginBoxInput}>
              <Form
                colon={false}
                name="basic"
                onFinish={onFinish}
              >
                <Form.Item
                  name="username"
                  rules={[
                    {
                      required: true,
                      message: "Please input your username",
                    },
                  ]}
                >
                  <Input prefix={<UserOutlined />} size="large" placeholder="Please input your username"/>
                </Form.Item>
                <Form.Item
                  name="password"
                  rules={[
                    {
                      required: true,
                      message:"Please input your password",
                    },
                  ]}
                >
                  <Input.Password prefix={<KeyOutlined />} size="large" placeholder="Please input your password"/>
                </Form.Item>
                <Form.Item>
                  <Button block type="primary" className={styles.loginBtn} htmlType="submit">Login</Button>
                </Form.Item>
              </Form>
          </div>
        </div>
      </div>
    );
  
}
export default connect(({ user: { userLogin } }) => ({
  userLogin,
}))(Login);

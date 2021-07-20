import React from "react";
import { PageLoading } from "@ant-design/pro-layout";
import { connect, history} from "umi";
class SecurityLayout extends React.Component {
  state = {
    isReady: false,
    login:true
  };

  componentDidMount() {
    this.setState({
      isReady: true
    });
    const { dispatch } = this.props;
    if (dispatch) {
      dispatch({
        type: "user/fetchCurrent"
      });
    }
  }
  render() {
    const { isReady, login} = this.state;
    const { children, loading, currentUser, ssoRedirect } = this.props; // You can replace it to your authentication rule (such as check token exists)
    // 你可以把它替换成你自己的登录认证规则（比如判断 token 是否存在）
    const isLogin = currentUser && currentUser.loginId;
   
    if ((!isLogin && loading) || !isReady) {
      return <PageLoading />;
    }
    
    if (ssoRedirect) {
      window.location.href = ssoRedirect;
      return null;
    }
   
    return children;
  }
}

export default connect(({ user, loading }) => ({
  currentUser: user.currentUser,
  ssoRedirect: user.ssoRedirect,
  loading: loading.models.user
}))(SecurityLayout);

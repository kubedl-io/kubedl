import { queryCurrentUser } from '@/services/global';
import { queryLogin, queryLoginOut } from '@/services/login';
import { history } from 'umi';
import { message } from 'antd';

const UserModel = {
  namespace: 'user',
  state: {
    currentUser: {},
    ssoRedirect: undefined,
    userLogin: {},
  },
  effects: {
    * fetchCurrent(_, { call, put }) {
      const response = yield call(queryCurrentUser);
      if (response.status === 401) {
        const payload = yield response.json();
        yield put({
          type: 'redirect',
          payload,
        });
      } else {
        yield put({
          type: 'saveCurrentUser',
          payload: response,
        });
      }
    },
    * fetchLogin({ payload }, { call, put }) {
      const response = yield call(queryLogin.bind(null, payload));
      if (response && response.status === 401) {

      } else {
        yield put({
          type: 'loginSuccess',
          payload: response || { data: { msg: null, success: null } },
        });
        
      }
    },
    * fetchLoginOut(_, { call, put }) {
      const response = yield call(queryLoginOut);
      if (response && response.status === 401) {
      } else {
        yield put({
          type: 'loginOut',
          payload: response || {},
        });
      }
    },
  },

  reducers: {
    redirect(state, action) {
      return { ...state, ssoRedirect: action.payload.data, currentUser: {} };
    },
    saveCurrentUser(state, action) {
      return {
        ...state,
        ssoRedirect: undefined,
        currentUser: action.payload.data || {},
      };
    },
    loginSuccess(state, { payload, payload: { data: { msg, success } } }) {
      if (success === 'true') {
        window.setTimeout(() => {
          history.push('/cluster');
        }, 2000);
      } else {
        message.error(msg || 'Login exception');
      }
      return { ...state, userLogin: payload };
    },
    loginOut(state, { payload }) {
      return { ...state, userLogin: payload };
    },
  },
};
export default UserModel;

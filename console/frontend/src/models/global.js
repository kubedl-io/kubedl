import { queryConfig, queryNamespaces } from "@/services/global";

const GlobalModel = {
  namespace: "global",
  state: {
    collapsed: false,
    config: undefined,
    notices: [],
    namespaces: [],
  },
  effects: {
    *fetchConfig(_, { call, put }) {
      let response;
      response = yield call(queryConfig);
      if (sessionStorage.getItem("namespace")) {
        response.data.namespace = sessionStorage.getItem("namespace");
      }
      yield put({
        type: "saveConfig",
        payload: response,
      });
    },
    *fetchNamespaces(_, { call, put }) {
      const response = yield call(queryNamespaces);
      const datatype = Object.prototype.toString.call(response.data);
      const namespaces = [];
      if (datatype === "[object Object]") {
        Object.entries(response.data).forEach((item) => {
          namespaces.push({
            value: item[0],
            label: `${item[0]}${item[1] ? "KubeDL" : ""}`,
          });
        });
      } else if (datatype === "[object Array]") {
        response.data.forEach((item) => {
          namespaces.push({
            label: item,
            value: item,
          });
        });
      }

      yield put({
        type: "saveNesponses",
        payload: namespaces,
      });
    },
  },
  reducers: {
    changeLayoutCollapsed(
      state = {
        notices: [],
        collapsed: true,
      },
      { payload }
    ) {
      return { ...state, collapsed: payload };
    },
    saveConfig(state, action) {
      return { ...state, config: action.payload.data };
    },
    saveNesponses(state, action) {
      return { ...state, namespaces: action.payload };
    },
  },
  subscriptions: {},
};
export default GlobalModel;

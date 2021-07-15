import slash from "slash2";
import defaultSettings from "./defaultSettings"; // preview.pro.ant.design only do not use in your production ;
import themePluginConfig from "./themePluginConfig"; // https://umijs.org/config/
// import  MonacoWebpackPlugin from 'monaco-editor-webpack-plugin';
const { pwa } = defaultSettings;
const HtmlWebpackPlugin = require("html-webpack-plugin");

const { ANT_DESIGN_PRO_ONLY_DO_NOT_USE_IN_YOUR_PRODUCTION } = process.env;
const plugins = [
  new HtmlWebpackPlugin({
    template: "index.html", // 需要生成html
  }),
  // 其他webpack plugin....
];
const isAntDesignProPreview =
  ANT_DESIGN_PRO_ONLY_DO_NOT_USE_IN_YOUR_PRODUCTION === "site";
if (isAntDesignProPreview) {
  // 针对 preview.pro.ant.design 的 GA 统计代码
  plugins.push([
    "umi-plugin-ga",
    {
      code: "UA-72788897-6",
    },
  ]);
  plugins.push(["umi-plugin-antd-theme", themePluginConfig]);
}
/** webpack.config.js */
const getPublicPath = () => {
  return "/";
};

export default {
  publicPath: getPublicPath(),
  dva: {},
  antd: {},
  hash: true,
  targets: {
    ie: 11,
  },
  locale: {
    default: "zh-CN",
    baseNavigator: true,
  },
  dynamicImport: {
    loading: "@/components/PageLoading/index",
  },
  alias: {
    "@components": "@/components",
  },
  routes: [
    {
      path: "/500",
      component: "./500",
    },
    {
      path: "/403",
      component: "./403",
    },
    {
      path: "/404",
      component: "./403",
    },
  ],
  theme: {
    // ...darkTheme,
  },
  define: {
    ANT_DESIGN_PRO_ONLY_DO_NOT_USE_IN_YOUR_PRODUCTION:
      ANT_DESIGN_PRO_ONLY_DO_NOT_USE_IN_YOUR_PRODUCTION || "", // preview.pro.ant.design only do not use in your production ; preview.pro.ant.design 专用环境变量，请不要在你的项目中使用它。
  },
  ignoreMomentLocale: true,
  lessLoader: {
    javascriptEnabled: true,
  },
  manifest: {
    basePath: "/",
  },
  proxy: {
    "/api/v1": {
      target: "http://localhost:9090/",
      changeOrigin: true,
      context(pathname, req) {
        return req.headers.accept === "application/json";
      },
    },
  },
};

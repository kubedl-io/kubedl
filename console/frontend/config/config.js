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
  const branch = process.env.BUILD_GIT_BRANCH;
  if (branch) {
    const { BUILD_GIT_GROUP } = process.env; // 例如："my-group"
    const { BUILD_GIT_PROJECT } = process.env; // 例如："my-project"
    const BUILD_GIT_VISION = branch.split("/")[1]; // 构建版本，例如："0.0.1"

    if (/^daily/i.test(branch)) {
      return `//dev.g.alicdn.com/${BUILD_GIT_GROUP}/${BUILD_GIT_PROJECT}/${BUILD_GIT_VISION}/`;
    }
    if (/^publish/i.test(branch)) {
      return `//g.alicdn.com/${BUILD_GIT_GROUP}/${BUILD_GIT_PROJECT}/${BUILD_GIT_VISION}/`;
    }
  }
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
      target: "http://100.83.223.159:9090/",
      changeOrigin: true,
      context(pathname, req) {
        // 对于登录后自动 302 的，不会带上 Accept header 但是也需要进行代理
        if (pathname.startsWith("/sendBucSSOToken.do")) {
          return true;
        }

        return req.headers.accept === "application/json";
      },
    },
  },
};

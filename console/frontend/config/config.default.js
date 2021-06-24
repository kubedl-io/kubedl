export default {
  define: {
    environment: "dlc"
  },
  routes: [
    {
      path: "/",
      component: "../layouts/SecurityLayout",
      routes: [
        {
          path: "/logIn",
          component: "./logIn"
        },
        {
          path: "/",
          component: "../layouts/BasicLayout",
          authority: ["admin", "user"],
          routes: [
            {
              path: "/",
              redirect: "/cluster"
            },
            {
              path: "/cluster",
              name: "cluster",
              icon: "home",
              component: "./ClusterInfo"
            },
            {
              path: "/jobs",
              name: "jobs",
              icon: "unordered-list",
              component: "./Jobs"
            },
            {
              path: "/datasheets",
              name: "datasheets",
              icon: "database",
              component: "./DataSheets"
            },
            {
              path: "/jobs/detail",
              component: "./JobDetail"
            },
            {
              path: "/jobs/job-create",
              component: "./JobCreate"
            },
            {
              path: "/job-submit",
              name: "job-submit",
              icon: "edit",
              component: "./JobCreate"
            },
            {
              path: "/datasheets/data-config",
              component: "./DataConfig"
            },
            {
              path: "/datasheets/git-config",
              component: "./GitConfig"
            },
            {
              component: "./404"
            }
          ]
        }
      ]
    }
  ]
};

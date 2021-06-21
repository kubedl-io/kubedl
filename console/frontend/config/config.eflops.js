export default {
  define: {
    environment: "eflops"
  },
  routes: [
    {
      path: "/logIn",
      component: "./logIn"
    },
    {
      path: "/",
      component: "../layouts/SecurityLayout",
      routes: [
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
              path: "/jobs/detail",
              component: "./JobDetail"
            },
            {
              path: "/jobs/job-create",
              component: "./JobCreate"
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

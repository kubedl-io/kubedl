export default {
  define: {
    environment: "kubedl"
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
              path: "/workspaces",
              name: "workspaces",
              icon: "unordered-list",
              component: "./Workspaces"
            },
            {
              path: "/workspace/detail",
              component: "./WorkspaceDetail"
            },
            {
              path: "/jobs",
              name: "jobs",
              icon: "unordered-list",
              component: "./Jobs"
            },
            {
              path: "/notebooks",
              name: "notebooks",
              icon: "unordered-list",
              component: "./Notebooks"
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
              path: "/notebook-submit",
              name: "notebook-submit",
              icon: "edit",
              component: "./NotebookCreate"
            },
            {
              path: "/datasheets/data-config",
              component: "./DataConfig"
            },
            {
              path: "/workspace-create",
              component: "./WorkspaceCreate"
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

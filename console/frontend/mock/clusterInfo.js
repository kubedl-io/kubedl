function getFakeCaptcha(req, res) {
  return res.json("captcha-xxx");
} // 代码中会兼容本地 service mock 以及部署站点的静态数据

export default {
  // 支持值为 Object 和 Array
  "GET /api/data/total": (req, res) => {
    res.send({
      code: "200",
      data: {
        totalCPU: 504000,
        totalMemory: 2747247054848,
        totalGPU: 16000
      }
    });
  },
  "GET /api/data/request/Running": (req, res) => {
    res.send({
      code: "200",
      data: {
        requestCPU: 12400,
        requestMemory: 40170946560,
        requestGPU: 2000
      }
    });
  },
  "GET /api/data/nodeInfos": (req, res) => {
    res.send({
      code: "200",
      data: {
        items: [
          {
            nodeName: "cn-hangzhou.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xx",
            accounted: 70
          },
          {
            nodeName: "cn-beijing.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xxx",
            accounted: 50
          },
          {
            nodeName: "cn-shanghai.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xxxx",
            accounted: 30
          },
          {
            nodeName: "cn-shenzhen.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xxxxx",
            accounted: 20
          },
          {
            nodeName: "cn-hangzhou.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xxxxxx",
            accounted: 10
          },
          {
            nodeName: "cn-hangzhou.192.174.35",
            instanceType: "ecs.gn5-c8g1.2xlarge",
            totalCPU: 96000,
            totalMemory: 776498278400,
            totalGPU: 8000,
            requestCPU: 400,
            requestMemory: 1493172224,
            requestGPU: 0,
            userName: "xxxxxx",
            accounted: 80
          }
        ],
        total: 10
      }
    });
  },
  "GET /api/v1/data/week/statistics": (req, res) => {
    res.send({
      code: "200",
      data: {
        items: [
          {
            userName: "xx",
            accounted: 70
          },
          {
            userName: "xxx",
            accounted: 50
          },
          {
            userName: "xxxx",
            accounted: 30
          },
          {
            userName: "xxxxx",
            accounted: 20
          },
          {
            userName: "xxxxxx",
            accounted: 10
          },
          {
            userName: "xxxxxx",
            accounted: 80
          }
        ],
        total: 10
      }
    });
  },
  "GET /api/v1/data/month/statistics": (req, res) => {
    res.send({
      code: "200",
      data: {
        items: [
          {
            userName: "test1",
            accounted: 70
          },
          {
            userName: "test2",
            accounted: 50
          },
          {
            userName: "test3",
            accounted: 30
          },
          {
            userName: "test4",
            accounted: 20
          },
          {
            userName: "test5",
            accounted: 10
          },
          {
            userName: "test6",
            accounted: 80
          }
        ],
        total: 10
      }
    });
  },
  "GET /api/v1/data/top/resources": (req, res) => {
    res.send({
      code: "200",
      data: {
        items: [
          {
            taskName: "top1",
            accounted: 8
          },
          {
            taskName: "top2",
            accounted: 20
          },
          {
            taskName: "top3",
            accounted: 10
          },
          {
            taskName: "top4",
            accounted: 40
          },
          {
            taskName: "top5",
            accounted: 30
          },
          {
            taskName: "top6",
            accounted: 5
          }
        ],
        total: 10
      }
    });
  }
};

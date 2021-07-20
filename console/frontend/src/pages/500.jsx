import { Button, Result } from 'antd';
import React from 'react';

const NoFoundPage = () => (
  <Result
    status="500"
    title="500"
    subTitle="系统发生异常，请返回重试"
    extra={
      <Button type="primary" onClick={() => window.location.href='/'}>
        返回
      </Button>
    }
  ></Result>
);

export default NoFoundPage;

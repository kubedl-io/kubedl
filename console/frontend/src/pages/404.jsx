import { Button, Result } from 'antd';
import React from 'react';

const NoFoundPage = () => (
  <Result
    status="404"
    title="404"
    subTitle="未找到相关页面"
    extra={
      <Button type="primary" onClick={() => window.location.href='/'}>
        返回
      </Button>
    }
  ></Result>
);

export default NoFoundPage;

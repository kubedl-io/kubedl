import { Button, Result } from 'antd';
import React from 'react';

const NoFoundPage = () => (
  <Result
    status="403"
    title="403"
    subTitle="无权访问，请返回重新登录"
    extra={
      <Button type="primary" onClick={() => window.location.href='/'}>
        返回
      </Button>
    }
  ></Result>
);

export default NoFoundPage;

import React, { Component, Fragment } from "react";
import { Modal, Input } from "antd";

const TextArea = Input.TextArea;

class SubmitModal extends React.Component {
  state = {
    loading: false,
    json: ""
  };

  componentDidMount() {}

  onChange(e) {
    this.setState({
      json: e.target.value
    });
  }

  handleSubmit = async e => {
    const { onCancel } = this.props;
    try {
      this.setState({
        loading: true
      });
      await submitJob(this.state.json);
      onCancel();
    } finally {
      this.setState({
        loading: false
      });
    }
  };

  handleCancel = e => {
    const { onCancel } = this.props;
    onCancel();
  };

  render() {
    return (
      <Modal
        width={1024}
        visible
        title="创建任务"
        onOk={this.handleSubmit}
        okText="提交"
        confirmLoading={this.state.loading}
        onCancel={this.handleCancel}
      >
        <TextArea rows={24} onChange={v => this.onChange(v)} />
      </Modal>
    );
  }
}
export default SubmitModal;

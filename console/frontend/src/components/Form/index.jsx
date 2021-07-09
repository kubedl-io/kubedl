import React, { Fragment, useEffect, useState } from "react";
import "antd/dist/antd.css";
import { Form, Select, Row, Col, Alert } from "antd";
import { MinusCircleOutlined, PlusOutlined } from "@ant-design/icons";
import { getLocale, useIntl } from "umi";
import styles from "../../pages/JobCreate/style.less";
export const FromAddDropDown = ({
  form,
  label,
  fieldCode,
  fieldKey,
  options = [],
  colStyle,
  onChange = () => {},
  alertMessage = {},
  messageLable,
  selectStyle = {
    "zh-CN": { paddingLeft: "1.22%" },
    "en-US": { paddingLeft: "2.88%" },
  },
  alertStyle = {
    "zh-CN": { flex: "0 0 77.7%", maxWidth: "77.7%" },
    "en-US": { flex: "0 0 60.7%", maxWidth: "60.7%" },
  },
}) => {
  const [alertMsg, setAlertMsg] = useState(alertMessage);
  const intl = useIntl();
  useEffect(() => {
    if (!form.getFieldValue(fieldCode)) {
      // 默认一列选择
      form.setFieldsValue({ [fieldCode]: [{ [fieldKey]: null }] });
    }
  }, []);
  useEffect(() => {
    if (form.getFieldValue(fieldCode)) {
      let optionGather = {};
      let msg = {};
      (options || []).forEach(({ name, local_path, pvc_name }) => {
        optionGather[name] = local_path;
        optionGather[`${name}&pvc_name`] = pvc_name;
      });
      form.getFieldValue(fieldCode).forEach(({ dataSource }, index) => {
        msg[dataSource] = optionGather[dataSource];
        msg[`${index}_pvc_name`] = optionGather[`${dataSource}&pvc_name`];
        msg[index] = dataSource;
      });
      setAlertMsg(msg);
    }
  }, [options]);
  const selChange = (oldIndex, value) => {
    let selOption =
      (options || []).filter(({ name }) => name === value)[0] || {};
    setAlertMsg((alertMsg) => {
      let msgGather = {
        ...alertMsg,
        [value]: selOption.local_path,
        [oldIndex]: value,
        [`${oldIndex}_pvc_name`]: selOption.pvc_name,
      };
      onChange(msgGather);
      return msgGather;
    });
  };
  return (
    <Form.List name={fieldCode}>
      {(fields, { add, remove }) => (
        <Fragment>
          {fields.map((field) => (
            <Form.Item noStyle shouldUpdate>
              {() => (
                <div>
                  <div
                    className={
                      getLocale() === "zh-CN"
                        ? styles.dataSourceContainer
                        : styles.dataSourceContainerEn
                    }
                  >
                    <Form.Item
                      {...field}
                      label={field.key === 0 ? label : " "}
                      colon={field.key === 0 ? true : false}
                      name={[field.name, fieldKey]}
                      fieldKey={[field.fieldKey, fieldKey]}
                      {...colStyle}
                    >
                      <Select
                        allowClear={true}
                        onChange={selChange.bind(null, field.key)}
                        style={selectStyle[getLocale()]}
                      >
                        {(options || []).map((option) => (
                          <Select.Option
                            key={option?.name}
                            value={option?.name}
                          >
                            {option?.name}
                          </Select.Option>
                        ))}
                      </Select>
                    </Form.Item>
                  </div>
                  <div className={styles.dataSourceButton}>
                    {field.key === 0 ? (
                      <PlusOutlined
                        onClick={add}
                        style={{ color: "#1890ff" }}
                      />
                    ) : (
                      <MinusCircleOutlined
                        style={{ color: "red", verticalAlign: "bottom" }}
                        onClick={() => {
                          let { [field.key]: removeKey, ...other } = alertMsg;
                          let msg = { ...other };
                          setAlertMsg(msg);
                          onChange(msg);
                          remove(field.name);
                        }}
                      />
                    )}
                  </div>
                  {JSON.stringify(alertMsg) !== "{}" && alertMsg[field.key] && (
                    <Row gutter={[24, 24]}>
                      <Col
                        span={getLocale() === "zh-CN" ? 18 : 14}
                        offset={getLocale() === "zh-CN" ? 4 : 8}
                        style={alertStyle[getLocale()]}
                      >
                        <Alert
                          type="info"
                          showIcon
                          message={
                            <span>
                              {messageLable} {": "}{" "}
                              {alertMsg[`${field.key}_pvc_name`]}
                              <br />
                              {intl.formatMessage({
                                id: "kubedl-dashboard-data-local-directory",
                              })}{" "}
                              {alertMsg[alertMsg[field.key]]}
                            </span>
                          }
                        />
                      </Col>
                    </Row>
                  )}
                </div>
              )}
            </Form.Item>
          ))}
        </Fragment>
      )}
    </Form.List>
  );
};

export const FormSel = ({
  form,
  name,
  label,
  rules,
  placeholder,
  onChange = () => {},
  listOption = [],
}) => {
  return (
    <Form.Item name={name} label={label} rules={rules}>
      <Select placeholder={placeholder} onChange={onChange} allowClear>
        {(listOption || []).map(({ label, value }) => {
          return (
            <Select.Option key={value} value={value}>
              {label}
            </Select.Option>
          );
        })}
      </Select>
    </Form.Item>
  );
};

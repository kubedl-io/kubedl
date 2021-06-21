import { QuestionCircleOutlined } from "@ant-design/icons";
import { Tooltip } from "antd";
import React from "react";
import { connect } from "umi";
import Avatar from "./AvatarDropdown";
import HeaderSearch from "../HeaderSearch";
import SelectLang from "../SelectLang";
import styles from "./index.less";

const GlobalHeaderRight = props => {
  const { theme, layout } = props;
  let className = styles.right;

  if (theme === "dark" && layout === "topmenu") {
    className = `${styles.right}  ${styles.dark}`;
  }

  return (
    <div className={className}>
      <Avatar />
      {/* <SelectLang className={styles.action} /> */}
    </div>
  );
};

export default connect(({ settings }) => ({
  theme: settings.navTheme,
  layout: settings.layout
}))(GlobalHeaderRight);

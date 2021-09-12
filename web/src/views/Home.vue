<template>
  <div>
    <a-layout id="components-layout-demo-fixed">
      <a-layout-content :style="{ padding: '0 0px', marginTop: '0px' }">
        <web-vulnerability></web-vulnerability>
      </a-layout-content>
      <!--请注意，如果您使用了本报告项目，请勿删除下方的链接。-->
      <!-- <div
        style="
          margin: 0px;
          padding: 0px;
          height: 28px;
          font-size: 18px;
          position: absolute;
          bottom: 0;
          left: 44.9%;
        "
      >
        <a style="color: #585858" href="https://xray.cool" target="_blank"
          >Modified From XRay Team</a
        >
      </div> -->
    </a-layout>
  </div>
</template>
<script>
import WebVulnerability from "../components/WebVulnerability";
export default {
  name: "Home",
  components: {
    WebVulnerability,
  },
};
</script>
<style lang="less">
.expand-detail *:not(.internal-detail) {
  .ant-descriptions-item-label {
    width: 150px;
  }
  .ant-descriptions-item-content,
  .ant-descriptions-item-label {
    border-bottom: 1px solid #e8e8e8;
  }
  pre {
    margin: 8px 0;
  }
  table {
    table-layout: fixed !important;
  }
}
body {
  background-color: #f0f2f5;
}
.filter-column {
  .anticon-filter {
    width: 32px !important;
    font-size: 14px !important;
    color: rgba(0, 0, 0, 0.85) !important;
    svg {
      margin-top: -8px !important;
      margin-left: -8px !important;
    }
  }
}
</style>

<template>
  <div class="mt-4">
    <SSLPageHeader
      @cloud-apply="openCloudApplyModal"
      @sync="openSyncModal"
      @upload="openUploadModal"
      @refresh="fetchData"
    />

    <n-card class="mt-8 rounded-3xl shadow-sm">
      <n-data-table
        :loading="loading"
        :columns="columns"
        :data="tableData"
        :bordered="false"
        :scroll-x="1200"
      />
    </n-card>

    <SSLCloudApplyModal
      v-model:show="cloudApplyModalVisible"
      :form="cloudApplyForm"
      :cloud-account-options="cloudAccountOptions"
      :cdn-domains-options="cdnDomainsOptions"
      :cdn-domains-loading="cdnDomainsLoading"
      :submitting="submitting"
      @cdn-account-change="handleCdnAccountChange"
      @submit="handleCloudApplyCertificate"
    />

    <SSLSyncModal
      v-model:show="syncModalVisible"
      :form="syncForm"
      :website-options="websiteOptions"
      :selected-runtime-text="selectedSyncWebsiteRuntimeText"
      :submitting="submitting"
      @submit="handleSyncCertificate"
    />

    <SSLUploadModal
      v-model:show="uploadModalVisible"
      :form="uploadForm"
      :submitting="submitting"
      @submit="handleUploadCertificate"
    />

    <SSLDetailModal
      v-model:show="detailModalVisible"
      :detail-title="detailTitle"
      :currentSsl="currentSSL"
      :build-bound-website-runtime-text="buildBoundWebsiteRuntimeText"
      @download="downloadContent"
    />

    <SSLApplyModal
      v-model:show="applyModalVisible"
      :form="applyForm"
      :website-options="websiteOptions"
      :selected-runtime-text="selectedApplyWebsiteRuntimeText"
      :submitting="submitting"
      @submit="handleApplyCertificate"
    />

    <SSLPushCdnModal
      v-model:show="pushCDNModalVisible"
      :form="pushCDNForm"
      :cloud-account-options="cloudAccountOptions"
      :currentSsl="currentPushSSL"
      :submitting="submitting"
      @submit="handlePushCDN"
    />

    <SSLPushRuleModal
      v-model:show="pushRuleModalVisible"
      :loading="pushRuleLoading"
      :submitting="submitting"
      :form="pushRuleForm"
      :data="pushRuleData"
      :columns="pushRuleColumns"
      :cloud-account-options="cloudAccountOptions"
      :current-primary-domain="currentPushSSL?.primaryDomain"
      @reset="resetPushRuleForm"
      @submit="handleSavePushRule"
    />

    <SSLLogModal
      :show="logModalVisible"
      :logs-data="logsData"
      @update:show="handleLogModalChange"
    />
  </div>
</template>

<script setup lang="ts">
import SSLApplyModal from "./components/SSLApplyModal.vue"
import SSLCloudApplyModal from "./components/SSLCloudApplyModal.vue"
import SSLDetailModal from "./components/SSLDetailModal.vue"
import SSLLogModal from "./components/SSLLogModal.vue"
import SSLPageHeader from "./components/SSLPageHeader.vue"
import SSLPushCdnModal from "./components/SSLPushCdnModal.vue"
import SSLPushRuleModal from "./components/SSLPushRuleModal.vue"
import SSLSyncModal from "./components/SSLSyncModal.vue"
import SSLUploadModal from "./components/SSLUploadModal.vue"
import { downloadContent } from "./components/sslHelpers"
import { useSSLManagement } from "./components/useSSLManagement"

const {
  loading,
  submitting,
  pushRuleLoading,
  cdnDomainsLoading,
  tableData,
  pushRuleData,
  cdnDomainsOptions,
  logsData,
  syncModalVisible,
  uploadModalVisible,
  detailModalVisible,
  applyModalVisible,
  pushCDNModalVisible,
  cloudApplyModalVisible,
  pushRuleModalVisible,
  logModalVisible,
  currentSSL,
  currentPushSSL,
  cloudApplyForm,
  pushCDNForm,
  pushRuleForm,
  syncForm,
  applyForm,
  uploadForm,
  websiteOptions,
  cloudAccountOptions,
  selectedSyncWebsiteRuntimeText,
  selectedApplyWebsiteRuntimeText,
  detailTitle,
  columns,
  pushRuleColumns,
  buildBoundWebsiteRuntimeText,
  fetchData,
  handleCdnAccountChange,
  resetPushRuleForm,
  openCloudApplyModal,
  openSyncModal,
  openUploadModal,
  handleLogModalChange,
  handleSavePushRule,
  handleSyncCertificate,
  handleCloudApplyCertificate,
  handleUploadCertificate,
  handleApplyCertificate,
  handlePushCDN
} = useSSLManagement()
</script>

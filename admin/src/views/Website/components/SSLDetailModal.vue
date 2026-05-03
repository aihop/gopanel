<script setup lang="ts">
import FullModal from "@/components/FullModal.vue"
import { formatDateTime, sourceLabel, type SSLRow } from "./sslHelpers"

defineProps<{
  show: boolean
  detailTitle: string
  currentSsl: SSLRow | null
  buildBoundWebsiteRuntimeText: (item: { id: number; name?: string; primaryDomain?: string }) => string
}>()

const emit = defineEmits<{
  (e: "update:show", value: boolean): void
  (e: "download", content: string, fileName: string): void
}>()
</script>

<template>
  <FullModal
    :show="show"
    :title="detailTitle"
    maxHeight="660px"
    @update:show="emit('update:show', $event)"
  >
    <div v-if="currentSsl" class="space-y-5">
      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-slate-400">{{ $t("website.primaryDomain") }}</div>
          <div class="mt-2 text-base font-semibold fg-base-100">{{ currentSsl.primaryDomain }}</div>
        </div>
        <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-slate-400">签发来源</div>
          <div class="mt-2 text-base font-semibold fg-base-100">
            <n-tag :type="sourceLabel(currentSsl).tagType" :bordered="false" round>
              {{ sourceLabel(currentSsl).label }}
            </n-tag>
          </div>
        </div>
        <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-slate-400">颁发者</div>
          <div class="mt-2 text-base font-semibold fg-base-100">{{ currentSsl.organization || "--" }}</div>
        </div>
        <div class="rounded-2xl border border-slate-200 bg-slate-50 p-4">
          <div class="text-xs uppercase tracking-[0.18em] text-slate-400">到期时间</div>
          <div class="mt-2 text-base font-semibold fg-base-100">{{ formatDateTime(currentSsl.expireDate) }}</div>
        </div>
      </div>

      <div class="grid grid-cols-1 gap-4 md:grid-cols-2">
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium text-slate-700">证书内容</div>
            <n-button text type="primary" @click="emit('download', currentSsl.pem, `${currentSsl.primaryDomain}.crt`)">
              下载 CRT
            </n-button>
          </div>
          <n-input :value="currentSsl.pem" type="textarea" readonly :autosize="{ minRows: 10, maxRows: 16 }" />
        </div>
        <div class="space-y-2">
          <div class="flex items-center justify-between">
            <div class="text-sm font-medium text-slate-700">私钥内容</div>
            <n-button text type="primary" @click="emit('download', currentSsl.privateKey, `${currentSsl.primaryDomain}.key`)">
              下载 KEY
            </n-button>
          </div>
          <n-input :value="currentSsl.privateKey" type="textarea" readonly :autosize="{ minRows: 10, maxRows: 16 }" />
        </div>
      </div>

      <div
        v-if="currentSsl.websites?.length"
        class="rounded-2xl border border-slate-200 bg-slate-50 p-4"
      >
        <div class="text-xs uppercase tracking-[0.18em] text-slate-400">绑定网站</div>
        <div class="mt-3 space-y-2">
          <div
            v-for="item in currentSsl.websites"
            :key="item.id"
            class="rounded-xl bg-white px-4 py-3 text-sm text-slate-600"
          >
            {{ buildBoundWebsiteRuntimeText(item) }}
          </div>
        </div>
      </div>
    </div>
  </FullModal>
</template>

<template>
  <div class="mobile-admin-switch">
    <button
      class="mobile-switch-btn"
      alt="mobile-admin-switch"
      aria-label="mobile-admin-switch"
      @click="showModal = true"
    >
      <Icon :size="20">
        <Iconify
          :icon="MobileIconHover"
          class="hover"
        />
        <Iconify :icon="MobileIcon" />
      </Icon>
    </button>

    <n-modal
      v-model:show="showModal"
      preset="card"
      style="width: 360px"
      title="手机管理"
    >
      <div class="flex flex-col items-center justify-center py-4">
        <p class="mb-4 text-center text-slate-500">
          请使用手机扫描下方二维码，
          <br />
          一键授权登录并关联服务器
        </p>
        <div class="rounded-xl border border-slate-100 bg-slate-50 p-4">
          <n-qr-code
            :value="qrCodeUrl"
            :size="200"
          />
        </div>
      </div>
    </n-modal>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from "vue"
import { NModal, NQrCode } from "naive-ui"
import Icon from "@/components/common/Icon.vue"
import { Icon as Iconify } from "@iconify/vue"

import { useAuthStore } from "@/store/auth"

const MobileIcon = "ion:phone-portrait-outline"
const MobileIconHover = "ion:phone-portrait"
const showModal = ref(false)

const authStore = useAuthStore()

// 这里可以根据实际需求生成一键授权的 URL
// 例如：结合当前时间戳、随机 token 或者请求后端生成的 session_id
const qrCodeUrl = computed(() => {
	const origin = window.location.origin
	return `${origin}/mobile/auth?token=${authStore.auth}`
})
</script>

<style scoped lang="scss">
.mobile-admin-switch {
	display: flex;
	align-items: center;
	justify-content: center;
}

.mobile-switch-btn {
	position: relative;
	width: 20px;
	height: 20px;
	overflow: hidden;
	outline: none;
	border: none;
	display: flex;
	align-items: center;
	justify-content: center;
	color: inherit;
	background: transparent;
	cursor: pointer;

	:deep() {
		.n-icon {
			position: absolute;
			top: 0;
			left: 0;
			transition: color 0.3s;

			& > svg {
				position: absolute;
				top: 0;
				left: 0;
				transition: opacity 0.35s;

				&.hover {
					opacity: 0;
				}
				&:not(.hover) {
					opacity: 1;
				}
			}

			&:hover {
				& > svg {
					&.hover {
						opacity: 1;
					}
					&:not(.hover) {
						opacity: 0;
					}
				}
			}
		}
	}
}
</style>

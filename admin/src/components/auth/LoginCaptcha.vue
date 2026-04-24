<template>
  <n-modal
    :show="visible"
    :mask-closable="true"
    transform-origin="center"
    @update:show="handleVisibleChange"
  >
    <div class="verify-card bg-base-100">
      <div class="verify-header">
        <h1 class="verify-title">
          {{ title }}
        </h1>
        <p class="verify-subtitle">
          {{ subTitle }}
        </p>
      </div>

      <div class="space-y-4">
        <div
          v-if="status === 'idle' || status === 'loading'"
          class="verify-box group"
          :class="{ 'is-loading': status === 'loading', 'is-idle': status === 'idle' }"
          @click="status === 'idle' && silentVerify()"
        >
          <div class="checkbox-wrapper">
            <div
              v-if="status === 'idle'"
              class="checkbox-idle"
            />
            <svg
              v-else
              class="spinner"
              fill="none"
              viewBox="0 0 24 24"
            >
              <circle
                style="opacity: 0.25"
                cx="12"
                cy="12"
                r="10"
                stroke="currentColor"
                stroke-width="4"
              />
              <path
                style="opacity: 0.75"
                fill="currentColor"
                d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4zm2 5.291A7.962 7.962 0 014 12H0c0 3.042 1.135 5.824 3 7.938l3-2.647z"
              />
            </svg>
          </div>
          <span class="verify-label">{{ loadingText }}</span>
          <svg
            class="arrow-icon"
            fill="none"
            viewBox="0 0 24 24"
          >
            <path
              fill="currentColor"
              d="M12 2l7 4v5c0 5.25-3.5 10-7 11-3.5-1-7-5.75-7-11V6l7-4z"
            />
            <path
              stroke="#fff"
              stroke-width="2"
              d="M9 12l2 2 4-4"
            />
          </svg>
        </div>

        <div
          v-show="status === 'manual'"
          class="manual-box"
        >
          <div class="manual-header">
            <span class="manual-title">{{ needVerifyText }}</span>
            <button
              class="manual-cancel"
              @click="reset"
            >
              {{ cancelText }}
            </button>
          </div>

          <div
            ref="containerRef"
            class="manual-img-container"
          >
            <img
              :src="imgBackSrc"
              class="manual-img"
              :class="{ hidden: !imgBackSrc }"
            />

            <img
              :src="imgBlockSrc"
              class="manual-block-img"
              :class="{ hidden: !imgBlockSrc }"
              :style="{
                top: `${pieceTop}px`,
                width: `${pieceWidth}px`,
                height: `${pieceHeight}px`,
                transform: `translateX(${moveBlockLeft}px)`
              }"
            />
          </div>

          <div class="slide-track">
            <div
              class="slide-progress"
              :style="{ width: `${moveBlockLeft + 25}px` }"
            />
            <div
              class="slide-thumb"
              :style="{ transform: `translateX(${moveBlockLeft}px)` }"
              @mousedown="startDrag"
              @touchstart="startDrag"
            >
              <svg
                class="slide-icon"
                fill="none"
                viewBox="0 0 16 16"
              >
                <path
                  fill="currentColor"
                  d="M5 4l4 4-4 4"
                  stroke="currentColor"
                  stroke-width="2"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </div>
            <div class="slide-text-container">
              <span class="slide-text">{{ slideText }}</span>
            </div>
          </div>
        </div>

        <div
          v-if="status === 'success'"
          class="success-box"
        >
          <div class="success-icon-wrapper">
            <svg
              class="success-icon"
              fill="none"
              viewBox="0 0 16 16"
            >
              <path
                d="M4 8l3 3 5-5"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </div>
          <div class="success-info">
            <p class="success-title">
              {{ successText }}
            </p>
            <p class="success-text">
              {{ brandText }}
            </p>
          </div>
        </div>
      </div>

      <div class="verify-footer">
        <div class="footer-line" />
        <span class="footer-brand">{{ brandText }}</span>
        <div class="footer-line" />
      </div>
    </div>
  </n-modal>
</template>

<script lang="ts" setup>
import CryptoJS from "crypto-js"
import { request } from "@/utils/network"
import { computed, ref } from "vue"
import { useThemeStore } from "@/store/theme"

type Status = "idle" | "loading" | "manual" | "success"

interface CaptchaGetResponse {
	originalImg: string
	blockImg: string
	token: string
	secretKey: string
	isOk: boolean
	pieceTop: number
	pieceWidth: number
	pieceHeight: number
}

interface CaptchaCheckResponse {
	token: string
}

interface ApiResponse<T> {
	code: number
	msg?: string
	data: T
}

const emit = defineEmits<{
	success: [token: string]
}>()

const themeStore = useThemeStore()
const visible = ref(false)
const status = ref<Status>("idle")
const token = ref("")
const secretKey = ref("")
const moveBlockLeft = ref(0)
const isDragging = ref(false)
const startX = ref(0)
const scaleRatio = ref(1)
const imgBackSrc = ref("")
const imgBlockSrc = ref("")
const pieceTop = ref(0)
const pieceWidth = ref(50)
const pieceHeight = ref(155)
const slideText = ref("拖动滑块完成验证")
const containerRef = ref<HTMLElement | null>(null)
const title = "安全验证"
const subTitle = "检测到登录风险，请先完成滑块验证"
const needVerifyText = "需要验证"
const cancelText = "取消"
const successText = "验证成功"
const brandText = "GoPanel Security"

const loadingText = computed(() => {
	return status.value === "loading" ? "验证资源加载中..." : "点击开始安全验证"
})

const isDark = computed(() => themeStore.isThemeDark)

function show() {
	reset()
	visible.value = true
}

function close() {
	visible.value = false
}

function handleVisibleChange(value: boolean) {
	visible.value = value
}

function reset() {
	status.value = "idle"
	moveBlockLeft.value = 0
	slideText.value = "拖动滑块完成验证"
}

async function silentVerify() {
	if (status.value === "success" || status.value === "loading") {
		return
	}

	status.value = "loading"
	try {
		const res = await request<ApiResponse<CaptchaGetResponse>>("/auth/verify/captcha/get", {
			method: "POST",
			data: { captchaType: "blockPuzzle" }
		})
		token.value = res.data.token
		secretKey.value = res.data.secretKey
		imgBackSrc.value = res.data.originalImg
		imgBlockSrc.value = res.data.blockImg
		pieceTop.value = res.data.pieceTop
		pieceWidth.value = res.data.pieceWidth
		pieceHeight.value = res.data.pieceHeight
		slideText.value = "拖动滑块完成验证"

		window.setTimeout(() => {
			status.value = res.data.isOk ? "success" : "manual"
			if (res.data.isOk) {
				handleSuccess()
			}
		}, 300)
	} catch (error) {
		console.error(error)
		status.value = "idle"
		slideText.value = "验证码加载失败，请重试"
	}
}

function handleSuccess() {
	status.value = "success"
	window.setTimeout(() => {
		emit("success", token.value)
		close()
	}, 500)
}

function getClientX(evt: MouseEvent | TouchEvent) {
	if (window.TouchEvent && evt instanceof TouchEvent) {
		return evt.changedTouches[0]?.clientX ?? 0
	}
	return (evt as MouseEvent).clientX
}

function startDrag(e: MouseEvent | TouchEvent) {
	if (window.TouchEvent && e instanceof TouchEvent && e.touches.length > 1) {
		return
	}
	if (e.cancelable && e.type === "touchstart") {
		e.preventDefault()
	}

	isDragging.value = true
	if (containerRef.value) {
		const rect = containerRef.value.getBoundingClientRect()
		scaleRatio.value = rect.width > 0 ? rect.width / 330 : 1
	}

	startX.value = getClientX(e)
	const startLogicLeft = moveBlockLeft.value

	const handleMove = (ev: MouseEvent | TouchEvent) => {
		if (!isDragging.value) {
			return
		}
		if (ev.cancelable && ev.type === "touchmove") {
			ev.preventDefault()
		}
		const currentX = getClientX(ev)
		const screenDelta = currentX - startX.value
		const logicDelta = screenDelta / scaleRatio.value
		moveBlockLeft.value = Math.max(0, Math.min(startLogicLeft + logicDelta, 280))
	}

	const handleEnd = () => {
		if (!isDragging.value) {
			return
		}
		isDragging.value = false
		window.removeEventListener("mousemove", handleMove)
		window.removeEventListener("mouseup", handleEnd)
		window.removeEventListener("touchmove", handleMove)
		window.removeEventListener("touchend", handleEnd)
		window.removeEventListener("touchcancel", handleEnd)
		submitManualVerify()
	}

	if (e.type === "mousedown") {
		window.addEventListener("mousemove", handleMove)
		window.addEventListener("mouseup", handleEnd)
	} else {
		window.addEventListener("touchmove", handleMove, { passive: false })
		window.addEventListener("touchend", handleEnd)
		window.addEventListener("touchcancel", handleEnd)
	}
}

function aesEncrypt(data: string, key: string) {
	const keyBytes = CryptoJS.enc.Utf8.parse(key)
	const encrypted = CryptoJS.AES.encrypt(data, keyBytes, {
		iv: keyBytes,
		mode: CryptoJS.mode.CBC,
		padding: CryptoJS.pad.Pkcs7
	})
	return encrypted.ciphertext
		.toString(CryptoJS.enc.Base64)
		.replace(/\+/g, "-")
		.replace(/\//g, "_")
}

async function submitManualVerify() {
	slideText.value = "验证中..."
	try {
		const encryptedPoint = aesEncrypt(JSON.stringify({ x: moveBlockLeft.value, y: 5 }), secretKey.value)
		const res = await request<ApiResponse<CaptchaCheckResponse>>("/auth/verify/captcha/check", {
			method: "POST",
			data: {
				captchaType: "blockPuzzle",
				point: encryptedPoint,
				token: token.value
			}
		})
		token.value = res.data.token
		handleSuccess()
	} catch (error) {
		console.error(error)
		moveBlockLeft.value = 0
		slideText.value = "验证失败，请重试"
		window.setTimeout(() => {
			status.value = "idle"
			silentVerify()
		}, 800)
	}
}

defineExpose({
	show,
	close
})
</script>

<style scoped>
.verify-card {
	width: 100%;
	max-width: 440px;
	border: 1px solid transparent;
	border-radius: 32px;
	padding: 2.5rem;
	box-shadow: 0 25px 50px -12px rgba(0, 0, 0, 0.25);
	transition: all 0.5s;
	margin: 0 auto;
}

.verify-header {
	text-align: center;
	margin-bottom: 2rem;
}

.verify-title {
	font-size: 1.5rem;
	font-weight: 800;
	color: v-bind("isDark ? '#f8fafc' : '#1d1d1f'");
	letter-spacing: 0.025em;
	margin: 0 0 0.5rem 0;
}

.verify-subtitle {
	font-size: 0.875rem;
	color: v-bind("isDark ? 'rgba(148,163,184,0.9)' : '#86868b'");
	margin: 0;
}

.space-y-4 > * + * {
	margin-top: 1rem;
}

.verify-box {
	display: flex;
	align-items: center;
	gap: 0.75rem;
	padding: 1rem;
	border: 1px solid #d2d2d7;
	border-radius: 1rem;
	transition: all 0.3s;
}

.verify-box.is-idle {
	cursor: pointer;
}

.verify-box.is-idle:hover {
	background-color: rgba(59, 130, 246, 0.08);
}

.verify-box.is-loading {
	cursor: not-allowed;
	opacity: 0.8;
}

.checkbox-wrapper {
	position: relative;
	width: 1.5rem;
	height: 1.5rem;
	display: flex;
	align-items: center;
	justify-content: center;
}

.checkbox-idle {
	width: 1.25rem;
	height: 1.25rem;
	border: 2px solid #d2d2d7;
	border-radius: 0.375rem;
	transition: all 0.3s;
}

.verify-box:hover .checkbox-idle {
	border-color: #3b82f6;
}

.spinner {
	animation: spin 1s linear infinite;
	height: 1.25rem;
	width: 1.25rem;
	color: #3b82f6;
}

@keyframes spin {
	from {
		transform: rotate(0deg);
	}

	to {
		transform: rotate(360deg);
	}
}

.verify-label {
	font-size: 0.875rem;
	font-weight: 500;
	color: v-bind("isDark ? '#e2e8f0' : '#424245'");
}

.arrow-icon {
	margin-left: auto;
	color: rgba(59, 130, 246, 0.3);
	transition: color 0.3s;
	width: 1.25rem;
	height: 1.25rem;
}

.verify-box:hover .arrow-icon {
	color: #3b82f6;
}

.manual-box {
	background-color: v-bind("isDark ? 'rgba(15,23,42,0.72)' : '#f5f5f7'");
	padding: 1.25rem;
	border-radius: 1rem;
	border: 1px solid v-bind("isDark ? 'rgba(71,85,105,0.8)' : '#e5e5e7'");
}

.manual-header {
	margin-bottom: 0.75rem;
	display: flex;
	justify-content: space-between;
	align-items: center;
}

.manual-title {
	font-size: 11px;
	font-weight: 700;
	color: #f97316;
	text-transform: uppercase;
	letter-spacing: -0.025em;
}

.manual-cancel {
	font-size: 10px;
	color: #94a3b8;
	cursor: pointer;
	background: none;
	border: none;
	padding: 0;
}

.manual-cancel:hover {
	color: #3b82f6;
}

.manual-img-container {
	position: relative;
	width: 330px;
	height: 155px;
	margin: 0 auto 1rem auto;
	background-color: #e2e8f0;
	border-radius: 0.75rem;
	overflow: hidden;
	border: 1px solid #cbd5e1;
}

.manual-img {
	width: 100%;
	height: 100%;
	object-fit: cover;
	display: block;
}

.manual-block-img {
	position: absolute;
	z-index: 20;
	left: 0;
	transition: none;
	will-change: transform;
}

.slide-track {
	background: v-bind("isDark ? 'rgba(15,23,42,0.86)' : '#f5f5f7'");
	border: 1px solid v-bind("isDark ? 'rgba(71,85,105,0.8)' : '#e5e5e7'");
	height: 42px;
	border-radius: 12px;
	position: relative;
}

.slide-progress {
	position: absolute;
	height: 100%;
	background-color: rgba(59, 130, 246, 0.2);
	border-radius: 0.75rem;
	transition: none;
}

.slide-thumb {
	position: absolute;
	left: 0;
	top: 0;
	width: 50px;
	height: 40px;
	background: #fff;
	border: 1px solid #d2d2d7;
	border-radius: 10px;
	cursor: pointer;
	box-shadow: 0 2px 5px rgba(0, 0, 0, 0.05);
	z-index: 10;
	display: flex;
	align-items: center;
	justify-content: center;
	touch-action: none !important;
	-webkit-touch-callout: none !important;
	user-select: none !important;
	-webkit-user-select: none !important;
}

.slide-icon {
	color: #94a3b8;
	width: 1rem;
	height: 1rem;
}

.slide-icon,
.slide-icon path {
	pointer-events: none;
}

.slide-text-container {
	position: absolute;
	top: 0;
	right: 0;
	bottom: 0;
	left: 0;
	display: flex;
	align-items: center;
	justify-content: center;
	pointer-events: none;
}

.slide-text {
	font-size: 10px;
	font-weight: 700;
	color: #94a3b8;
	text-transform: uppercase;
	letter-spacing: 0.2em;
}

.success-box {
	display: flex;
	align-items: center;
	gap: 1rem;
	padding: 1.25rem;
	background-color: #ecfdf5;
	border: 1px solid #d1fae5;
	border-radius: 1rem;
}

.success-icon-wrapper {
	width: 1.75rem;
	height: 1.75rem;
	background-color: #10b981;
	border-radius: 50%;
	display: flex;
	align-items: center;
	justify-content: center;
	color: white;
	font-size: 0.75rem;
	box-shadow: 0 10px 15px -3px rgba(167, 243, 208, 0.5);
}

.success-icon {
	width: 1rem;
	height: 1rem;
}

.success-info p {
	margin: 0;
}

.success-title {
	font-size: 0.875rem;
	font-weight: 700;
	color: #065f46;
}

.success-text {
	font-size: 10px;
	color: rgba(5, 150, 105, 0.7);
	text-transform: uppercase;
	font-family:
		ui-monospace,
		SFMono-Regular,
		Menlo,
		Monaco,
		Consolas,
		"Liberation Mono",
		"Courier New",
		monospace;
}

.verify-footer {
	margin-top: 2rem;
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 0.75rem;
	opacity: 0.3;
}

.footer-line {
	height: 1px;
	width: 2rem;
	background-color: #94a3b8;
}

.footer-brand {
	font-size: 9px;
	font-weight: 900;
	text-transform: uppercase;
	letter-spacing: 0.1em;
	color: v-bind("isDark ? '#f8fafc' : '#0f172a'");
}

.hidden {
	display: none !important;
}

@media (max-width: 480px) {
	.verify-card {
		padding: 1.25rem;
		border-radius: 20px;
	}

	.manual-box {
		padding: 0.75rem;
		width: 100%;
		overflow-x: hidden;
		display: flex;
		flex-direction: column;
		align-items: center;
	}

	.manual-header {
		width: 100%;
		max-width: 330px;
	}

	.manual-img-container,
	.slide-track {
		width: 330px !important;
		transform-origin: center top;
		transform: scale(0.88);
		margin-left: 0 !important;
		margin-right: 0 !important;
	}

	.manual-img-container {
		margin-bottom: -10px;
	}

	.slide-track {
		margin-bottom: -5px;
	}
}

@media (max-width: 360px) {
	.manual-img-container,
	.slide-track {
		transform: scale(0.75);
	}

	.manual-img-container {
		margin-bottom: -25px;
	}
}
</style>

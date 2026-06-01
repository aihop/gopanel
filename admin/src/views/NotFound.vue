<template>
  <div class="not-found-page">
    <div class="content-wrapper">
      <!-- 背景光效 -->
      <div class="glow-bg blue"></div>
      <div class="glow-bg purple"></div>

      <!-- 核心内容 -->
      <div class="error-container">
        <div class="error-code">
          <span class="gradient-text">4</span>
          <div class="orbit-circle">
            <div class="satellite"></div>
          </div>
          <span class="gradient-text">4</span>
        </div>

        <h1 class="title">页面在星海中迷失了方向</h1>
        <p class="desc">您访问的页面可能已经被移除、重命名，或者您输入的地址有误。<br>请检查地址是否正确，或返回控制台首页。</p>

        <div class="action-buttons">
          <n-button
            size="large"
            type="primary"
            class="home-btn"
            @click="redirect()"
          >
            返回控制台首页
          </n-button>
          <n-button
            size="large"
            ghost
            class="back-btn"
            @click="goBack()"
          >
            返回上一页
          </n-button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { NButton } from "naive-ui"
import { useRouter } from "vue-router"

const router = useRouter()

function redirect() {
	router.push({ path: "/" })
}

function goBack() {
	router.back()
}
</script>

<style lang="scss" scoped>
.not-found-page {
	height: 100vh;
	width: 100vw;
	background-color: var(--bg-body-color);
	display: flex;
	align-items: center;
	justify-content: center;
	overflow: hidden;
	position: relative;
	background-image: radial-gradient(var(--border-color) 1px, transparent 1px);
	background-size: 24px 24px;
}

.content-wrapper {
	position: relative;
	width: 100%;
	max-width: 1200px;
	display: flex;
	justify-content: center;
	z-index: 10;
}

/* 渐变氛围光效 */
.glow-bg {
	position: absolute;
	width: 600px;
	height: 600px;
	border-radius: 50%;
	filter: blur(100px);
	opacity: 0.4;
	z-index: -1;
	animation: pulse 8s cubic-bezier(0.4, 0, 0.2, 1) infinite alternate;

	&.blue {
		background: linear-gradient(135deg, #3b82f6, #60a5fa);
		top: -200px;
		left: -100px;
	}
	&.purple {
		background: linear-gradient(135deg, #8b5cf6, #c084fc);
		bottom: -200px;
		right: -100px;
		animation-delay: -4s;
	}
}

.error-container {
	text-align: center;
	padding: 40px;
	background: color-mix(in srgb, var(--bg-default-color) 70%, transparent);
	backdrop-filter: blur(20px);
	-webkit-backdrop-filter: blur(20px);
	border: 1px solid color-mix(in srgb, var(--border-color) 80%, transparent);
	border-radius: 32px;
	box-shadow: 0 24px 48px rgba(15, 23, 42, 0.05);
	transform: translateY(20px);
	opacity: 0;
	animation: floatUp 0.8s cubic-bezier(0.16, 1, 0.3, 1) forwards;
}

.error-code {
	display: flex;
	align-items: center;
	justify-content: center;
	gap: 16px;
	font-size: 160px;
	font-weight: 800;
	line-height: 1;
	margin-bottom: 24px;
	font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
}

.gradient-text {
	background: linear-gradient(135deg, var(--fg-default-color) 0%, var(--fg-secondary-color) 100%);
	-webkit-background-clip: text;
	-webkit-text-fill-color: transparent;
}

/* 轨道动画 0 */
.orbit-circle {
	width: 140px;
	height: 140px;
	border: 8px solid color-mix(in srgb, var(--primary-color) 15%, var(--bg-secondary-color));
	border-radius: 50%;
	position: relative;
	box-shadow: inset 0 0 20px rgba(59, 130, 246, 0.1);
	background: linear-gradient(135deg, color-mix(in srgb, var(--bg-default-color) 80%, transparent), color-mix(in srgb, var(--bg-default-color) 40%, transparent));
	
	&::after {
		content: '';
		position: absolute;
		top: 50%;
		left: 50%;
		transform: translate(-50%, -50%);
		width: 60px;
		height: 60px;
		background: linear-gradient(135deg, #3b82f6, #8b5cf6);
		border-radius: 50%;
		box-shadow: 0 8px 16px rgba(59, 130, 246, 0.3);
	}
}

.satellite {
	position: absolute;
	top: -4px;
	left: 50%;
	width: 16px;
	height: 16px;
	background: #3b82f6;
	border-radius: 50%;
	box-shadow: 0 0 10px #3b82f6;
	transform-origin: 0 74px; /* 轨道半径 */
	animation: orbit 3s linear infinite;
}

.title {
	font-size: 32px;
	font-weight: 700;
	color: var(--fg-default-color);
	margin-bottom: 16px;
	letter-spacing: -0.02em;
}

.desc {
	font-size: 16px;
	color: var(--fg-secondary-color);
	line-height: 1.6;
	margin-bottom: 40px;
}

.action-buttons {
	display: flex;
	gap: 16px;
	justify-content: center;

	.home-btn {
		--n-border-radius: 16px;
		padding: 0 32px;
		font-weight: 600;
		box-shadow: 0 8px 20px rgba(59, 130, 246, 0.2);
		transition: transform 0.2s, box-shadow 0.2s;

		&:hover {
			transform: translateY(-2px);
			box-shadow: 0 12px 24px rgba(59, 130, 246, 0.3);
		}
	}

	.back-btn {
		--n-border-radius: 16px;
		padding: 0 32px;
		font-weight: 600;
		transition: transform 0.2s;

		&:hover {
			transform: translateY(-2px);
		}
	}
}

@keyframes orbit {
	100% { transform: rotate(360deg); }
}

@keyframes pulse {
	0% { transform: scale(1); opacity: 0.4; }
	100% { transform: scale(1.1); opacity: 0.6; }
}

@keyframes floatUp {
	100% { transform: translateY(0); opacity: 1; }
}

@media (max-width: 768px) {
	.error-code {
		font-size: 100px;
		.orbit-circle {
			width: 90px;
			height: 90px;
			border-width: 6px;
			&::after {
				width: 40px;
				height: 40px;
			}
		}
		.satellite {
			transform-origin: 0 49px;
			width: 12px;
			height: 12px;
			top: -3px;
		}
	}
	.title { font-size: 24px; }
	.desc { font-size: 14px; }
	.action-buttons {
		flex-direction: column;
		.n-button { width: 100%; }
	}
}
</style>

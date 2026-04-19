<script setup lang="ts">
import { ref, computed } from "vue"
import { NModal, NForm, NFormItem, NInput, NSelect, NButton, NRadioGroup, NRadio, NCheckboxGroup, NCheckbox, useMessage } from "naive-ui"

const props = defineProps<{
  show: boolean
  templateType: string // 'php', 'node', 'go', 'java'
}>()

const emit = defineEmits(["update:show", "generate"])

const message = useMessage()

// PHP 配置
const phpVersion = ref("8.2")
const phpExtensions = ref(["gd", "zip", "pdo_mysql"])
const phpComposer = ref(true)
const phpFramework = ref("laravel")

// Node.js 配置
const nodeVersion = ref("18")
const nodePackageManager = ref("npm")
const nodeFramework = ref("vue")

// Go 配置
const goVersion = ref("1.21")
const goProxy = ref("https://goproxy.cn,direct")

// Java 配置
const javaVersion = ref("17")
const javaTool = ref("maven")

const phpVersionOptions = [
  { label: "PHP 8.3", value: "8.3" },
  { label: "PHP 8.2", value: "8.2" },
  { label: "PHP 8.1", value: "8.1" },
  { label: "PHP 8.0", value: "8.0" },
  { label: "PHP 7.4", value: "7.4" }
]

const phpExtOptions = [
  { label: "GD (图像处理)", value: "gd" },
  { label: "Zip (压缩解压)", value: "zip" },
  { label: "PDO MySQL (数据库)", value: "pdo_mysql" },
  { label: "Redis (缓存)", value: "redis" },
  { label: "BCMath (精确计算)", value: "bcmath" },
  { label: "Opcache (性能优化)", value: "opcache" }
]

const nodeVersionOptions = [
  { label: "Node.js 20 (LTS)", value: "20" },
  { label: "Node.js 18 (LTS)", value: "18" },
  { label: "Node.js 16", value: "16" }
]

const handleClose = () => {
  emit("update:show", false)
}

const handleGenerate = () => {
  let templateConfig: any = {}

  if (props.templateType === 'php') {
    let script = `apk add --no-cache git unzip \n`
    
    // 简化的扩展安装脚本示意，实际使用官方 docker-php-ext-install
    if (phpExtensions.value.length > 0) {
      script += `# 安装 PHP 扩展: ${phpExtensions.value.join(', ')}\n`
      script += `docker-php-ext-install ${phpExtensions.value.filter(ext => !['redis'].includes(ext)).join(' ')} \n`
      if (phpExtensions.value.includes('redis')) {
        script += `pecl install redis && docker-php-ext-enable redis \n`
      }
    }

    if (phpComposer.value) {
      script += `\n# 安装 Composer\ncurl -sS https://getcomposer.org/installer | php -- --install-dir=/usr/local/bin --filename=composer\n`
      script += `composer install --no-dev --optimize-autoloader\n`
    }

    if (phpFramework.value === 'laravel') {
      script += `\n# Laravel 特有构建步骤\ncp .env.example .env\nphp artisan key:generate\nphp artisan config:cache\n`
    }

    templateConfig = {
      name: `PHP ${phpVersion.value} 部署流水线`,
      description: `自动化部署 ${phpFramework.value.toUpperCase()} 项目 (PHP ${phpVersion.value})`,
      buildEnv: "docker",
      buildImage: `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/php:${phpVersion.value}-cli-alpine`,
      buildScript: script,
      artifactPath: "./",
    }
  } 
  else if (props.templateType === 'node') {
    let script = ""
    if (nodePackageManager.value === 'npm') {
      script = `npm install\nnpm run build`
    } else if (nodePackageManager.value === 'yarn') {
      script = `yarn install\nyarn build`
    } else if (nodePackageManager.value === 'pnpm') {
      script = `npm install -g pnpm\npnpm install\npnpm build`
    }

    let artifact = "dist/"
    if (nodeFramework.value === 'nextjs') artifact = ".next/"
    else if (nodeFramework.value === 'nuxtjs') artifact = ".output/"

    templateConfig = {
      name: `Node.js ${nodeVersion.value} 部署流水线`,
      description: `构建 ${nodeFramework.value.toUpperCase()} 前端项目 (Node ${nodeVersion.value})`,
      buildEnv: "docker",
      buildImage: `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/node:${nodeVersion.value}-alpine`,
      buildScript: script,
      artifactPath: artifact,
    }
  }
  else if (props.templateType === 'go') {
    templateConfig = {
      name: `Go ${goVersion.value} 部署流水线`,
      description: "编译构建 Go 项目可执行文件",
      buildEnv: "docker",
      buildImage: `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/golang:${goVersion.value}-alpine`,
      buildScript: `go env -w GO111MODULE=on\ngo env -w GOPROXY=${goProxy.value}\ngo build -o app main.go`,
      artifactPath: "app",
    }
  }
  else if (props.templateType === 'java') {
    templateConfig = {
      name: `Java ${javaVersion.value} 部署流水线`,
      description: `使用 ${javaTool.value.toUpperCase()} 构建 Java 项目`,
      buildEnv: "docker",
      buildImage: javaTool.value === 'maven' 
        ? `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/maven:3.9-eclipse-temurin-${javaVersion.value}-alpine`
        : `swr.cn-north-4.myhuaweicloud.com/ddn-k8s/docker.io/library/gradle:8-jdk${javaVersion.value}-alpine`,
      buildScript: javaTool.value === 'maven' ? `mvn clean package -DskipTests` : `gradle build -x test`,
      artifactPath: javaTool.value === 'maven' ? "target/*.jar" : "build/libs/*.jar",
    }
  }

  handleClose()
  emit("generate", templateConfig)
}

const modalTitle = computed(() => {
  switch (props.templateType) {
    case 'php': return "PHP 环境模板向导"
    case 'node': return "Node.js 环境模板向导"
    case 'go': return "Go 环境模板向导"
    case 'java': return "Java 环境模板向导"
    default: return "环境模板向导"
  }
})
</script>

<template>
  <n-modal
    :show="show"
    @update:show="handleClose"
    preset="card"
    :title="modalTitle"
    style="width: 600px"
    class="rounded-xl"
  >
    <div class="px-2 py-4">
      <!-- PHP 配置 -->
      <n-form v-if="templateType === 'php'" label-placement="left" label-width="120">
        <n-form-item label="PHP 版本">
          <n-select v-model:value="phpVersion" :options="phpVersionOptions" />
        </n-form-item>
        <n-form-item label="项目框架">
          <n-radio-group v-model:value="phpFramework">
            <n-radio value="laravel">Laravel</n-radio>
            <n-radio value="thinkphp">ThinkPHP</n-radio>
            <n-radio value="other">其他/原生</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="使用 Composer">
          <n-radio-group v-model:value="phpComposer">
            <n-radio :value="true">是 (执行 install)</n-radio>
            <n-radio :value="false">否</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="常用扩展">
          <n-checkbox-group v-model:value="phpExtensions">
            <div class="grid grid-cols-2 gap-2">
              <n-checkbox v-for="ext in phpExtOptions" :key="ext.value" :value="ext.value" :label="ext.label" />
            </div>
          </n-checkbox-group>
        </n-form-item>
      </n-form>

      <!-- Node.js 配置 -->
      <n-form v-if="templateType === 'node'" label-placement="left" label-width="120">
        <n-form-item label="Node 版本">
          <n-select v-model:value="nodeVersion" :options="nodeVersionOptions" />
        </n-form-item>
        <n-form-item label="包管理器">
          <n-radio-group v-model:value="nodePackageManager">
            <n-radio value="npm">NPM</n-radio>
            <n-radio value="yarn">Yarn</n-radio>
            <n-radio value="pnpm">PNPM</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="前端框架">
          <n-radio-group v-model:value="nodeFramework">
            <n-radio value="vue">Vue / React (dist)</n-radio>
            <n-radio value="nextjs">Next.js</n-radio>
            <n-radio value="nuxtjs">Nuxt.js</n-radio>
          </n-radio-group>
        </n-form-item>
      </n-form>

      <!-- Go 配置 -->
      <n-form v-if="templateType === 'go'" label-placement="left" label-width="120">
        <n-form-item label="Go 版本">
          <n-input v-model:value="goVersion" placeholder="例如: 1.21" />
        </n-form-item>
        <n-form-item label="GOPROXY 镜像">
          <n-input v-model:value="goProxy" placeholder="https://goproxy.cn,direct" />
        </n-form-item>
      </n-form>

      <!-- Java 配置 -->
      <n-form v-if="templateType === 'java'" label-placement="left" label-width="120">
        <n-form-item label="Java 版本">
          <n-radio-group v-model:value="javaVersion">
            <n-radio value="17">JDK 17</n-radio>
            <n-radio value="11">JDK 11</n-radio>
            <n-radio value="8">JDK 8</n-radio>
          </n-radio-group>
        </n-form-item>
        <n-form-item label="构建工具">
          <n-radio-group v-model:value="javaTool">
            <n-radio value="maven">Maven</n-radio>
            <n-radio value="gradle">Gradle</n-radio>
          </n-radio-group>
        </n-form-item>
      </n-form>
      
      <div class="mt-4 p-3 bg-blue-50 text-blue-600 rounded-lg text-sm">
        💡 提示：生成模板后，您依然可以在“构建脚本”中自由修改任何执行细节。
      </div>
    </div>
    
    <template #footer>
      <div class="flex justify-end gap-3">
        <n-button @click="handleClose">取消</n-button>
        <n-button type="primary" @click="handleGenerate">一键生成流水线</n-button>
      </div>
    </template>
  </n-modal>
</template>
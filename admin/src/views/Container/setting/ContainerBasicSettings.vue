<template>
  <n-spin :show="daemonLoading">
    <n-form
      class="space-y-6 p-4"
      label-placement="left"
      label-width="100px"
      :style="{ maxWidth: '640px' }"
    >
      <n-form-item label="镜像加速">
        <div>
          <div class="flex items-end gap-2">
            <n-input
              v-if="daemon.registryMirrors"
              type="textarea"
              :disabled="mirrorsDisabled"
              :value="daemon.registryMirrors.join('\n')"
              class="min-h-[34px]"
              placeholder="https://dockerpull.pw&#10;https://dockerhub.icu&#10;https://hub.rat.dev&#10;https://register.librax.org&#10;https://docker-0.unsee.tech"
              :autosize="{ minRows: 1, maxRows: 5 }"
              @update:value="emit('update-mirror-urls', $event, 'registryMirrors')"
            />
            <n-button
              :disabled="mirrorsDisabled"
              @click="emit('open-drawer', 'registryMirrors')"
            >
              <template #icon>
                <Icon name="uil:setting" />
              </template>
              设置
            </n-button>
          </div>
          <div class="mt-1 text-xs leading-7 text-gray-500">
            {{ $t("container.mirrorsHelper") }}
          </div>
        </div>
      </n-form-item>

      <n-form-item label="日志切割">
        <n-spin :show="logPruneLoading">
          <n-switch
            :value="logSwitchValue"
            :disabled="dockerOnly"
            @update:value="emit('log-switch-change', $event)"
          />
          <template v-if="logSwitchValue">
            <n-space class="mt-2">
              <n-tag type="info">单文件最大: {{ daemon.logMaxSize }}</n-tag>
              <n-tag type="info">最大文件数: {{ daemon.logMaxFile }}</n-tag>
            </n-space>
          </template>
        </n-spin>
      </n-form-item>

      <n-form-item label="iptables">
        <div class="w-full">
          <n-switch
            :value="daemon.iptables"
            :disabled="dockerOnly"
            @update:value="emit('iptables-change', $event)"
          />
          <n-text depth="3" class="mt-1 block text-xs">Docker 对 iptables 规则的自动配置</n-text>
        </div>
      </n-form-item>

      <n-form-item label="Live restore">
        <div class="w-full">
          <n-switch
            :value="daemon.liveRestore"
            :disabled="dockerOnly"
            @update:value="emit('live-restore-change', $event)"
          />
          <n-text depth="3" class="mt-1 block text-xs">
            允许在 Docker 守护进程发生意外停机或崩溃时保留正在运行的容器状态
          </n-text>
        </div>
      </n-form-item>

      <n-form-item label="cgroup driver">
        <n-radio-group
          :value="daemon.cgroupDriver"
          name="cgroupdriver"
          :disabled="dockerOnly"
          @update:value="emit('cgroup-driver-change', $event)"
        >
          <n-radio-button value="cgroupfs" label="cgroupfs" />
          <n-radio-button value="systemd" label="systemd" />
        </n-radio-group>
      </n-form-item>
    </n-form>
  </n-spin>
</template>

<script setup lang="ts">
import { computed } from "vue"

const props = defineProps<{
  daemon: any
  validate: any
  dockerOnly: boolean
  daemonLoading: boolean
  logPruneLoading: boolean
  logSwitchValue: boolean
}>()

const emit = defineEmits<{
  (e: "update-mirror-urls", value: string, key: string): void
  (e: "open-drawer", key: string): void
  (e: "log-switch-change", value: boolean): void
  (e: "iptables-change", value: boolean): void
  (e: "live-restore-change", value: boolean): void
  (e: "cgroup-driver-change", value: string): void
}>()

const mirrorsDisabled = computed(
  () =>
    !(
      props.daemon.capabilities?.daemonJson ||
      props.daemon.capabilities?.podmanRegistriesConf ||
      (props.validate?.os === "linux" && props.validate?.runtimeKind === "podman" && props.validate?.gpc?.reachable)
    )
)
</script>

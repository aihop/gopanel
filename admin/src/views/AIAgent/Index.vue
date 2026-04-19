<template>
  <div
    class="bg-base-accent border-base-accent rounded-[28px] w-full relative"
    style="min-height: calc(100vh - 130px); height: 100%; display: flex; flex-direction: column;"
  >
    <div class="group-lobby flex-1 overflow-y-auto p-6 md:p-10">
      <div class="lobby-header mb-10 flex justify-between items-center">
        <div>
          <h1 class="text-2xl font-bold text-[var(--n-text-color)] mb-2">{{ $t('ai.workspace') }}</h1>
          <p class="text-[var(--n-text-color-3)] text-sm">{{ $t('ai.workspaceDesc') }}</p>
        </div>
        <n-button
          type="primary"
          size="large"
          @click="showCreateGroupModal = true"
          round
        >
          <template #icon>
            <AddIcon />
          </template>
          {{ $t('ai.createGroup') }}
        </n-button>
      </div>

      <div
        v-if="groups.length === 0"
        class="flex justify-center items-center h-64"
      >
        <n-empty
          :description="$t('ai.noGroup')"
          size="huge"
        />
      </div>

      <div
        v-else
        class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6"
      >
        <div
          v-for="group in groups"
          :key="group.id"
          class="group-card cursor-pointer relative overflow-hidden rounded-3xl p-6"
          @click="enterGroup(group.id)"
        >
          <div class="group-card__glow"></div>
          <div class="group-card__grid"></div>
          <div class="flex justify-between items-start mb-4">
            <div class="group-card__avatar w-12 h-12 rounded-2xl flex items-center justify-center text-xl font-bold">
              {{ group.name.substring(0, 1).toUpperCase() }}
            </div>
            <n-tag
              size="small"
              type="info"
              round
            >{{ group.memberCount || 1 }} {{ $t('ai.member') }}</n-tag>
          </div>
          <h3 class="group-card__title text-lg font-semibold mb-2">{{ group.name }}</h3>
          <p class="group-card__desc text-sm line-clamp-2 mb-5">{{ group.description || $t('ai.noDesc') }}</p>
          <div class="group-card__footer flex justify-between items-center text-xs pt-4">
            <span>{{ group.taskCount || 0 }} {{ $t('ai.task') }}</span>
            <span class="group-card__action">{{ $t('ai.enterWorkspace') }}</span>
          </div>
        </div>
      </div>
      <n-modal
        v-model:show="showCreateGroupModal"
        preset="dialog"
        :title="$t('ai.createGroup')"
      >
        <div class="flex flex-col gap-4 mt-4">
          <n-input
            v-model:value="newGroupForm.name"
            :placeholder="$t('ai.groupName')"
            placeholder-class="text-[var(--n-text-color-3)]"
          />
          <n-input
            v-model:value="newGroupForm.desc"
            type="textarea"
            :placeholder="$t('ai.groupDesc')"
          />
        </div>
        <template #action>
          <n-button @click="showCreateGroupModal = false">{{ $t('commons.button.cancel') }}</n-button>
          <n-button
            type="primary"
            @click="submitCreateGroup"
          >{{ $t('commons.button.confirm') }}</n-button>
        </template>
      </n-modal>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMessage } from 'naive-ui'
import { getAIGroups, createAIGroup } from '@/api/modules/ai_agent'
import type { AIGroup } from '@/api/interface/ai_agent'

const AddIcon = () => '+'

const message = useMessage()
const router = useRouter()

const showCreateGroupModal = ref(false)
const newGroupForm = ref({ name: '', desc: '' })

const groups = ref<AIGroup[]>([])

const fetchGroups = async () => {
  try {
    const res = await getAIGroups({ page: 1, pageSize: 50 })
    if (res.code === 0) {
      groups.value = res.data.items || []
    }
  } catch (error) {
    console.error('获取项目组失败:', error)
  }
}

onMounted(() => {
  fetchGroups()
})

const submitCreateGroup = async () => {
  if (!newGroupForm.value.name.trim()) {
    message.warning('组名称不能为空')
    return
  }
  try {
    const res = await createAIGroup({
      name: newGroupForm.value.name,
      description: newGroupForm.value.desc
    })
    if (res.code === 0) {
      showCreateGroupModal.value = false
      newGroupForm.value = { name: '', desc: '' }
      message.success('创建成功，可以邀请组员加入了！')
      fetchGroups()
    }
  } catch (error) {
    message.error('创建项目组失败')
  }
}

const enterGroup = (id: number) => {
  router.push(`/ai/group/${id}`)
}
</script>

<style scoped>
.group-card {
  background:
    radial-gradient(circle at top right, color-mix(in srgb, var(--n-primary-color) 16%, transparent), transparent 34%),
    linear-gradient(180deg, color-mix(in srgb, var(--n-color) 98%, white 2%), color-mix(in srgb, var(--n-color) 94%, black 6%));
  border: 1px solid color-mix(in srgb, var(--n-border-color) 88%, var(--n-primary-color) 12%);
  box-shadow:
    0 12px 28px rgba(15, 23, 42, 0.06),
    inset 0 1px 0 rgba(255, 255, 255, 0.5);
  transition:
    transform 0.28s ease,
    box-shadow 0.28s ease,
    border-color 0.28s ease,
    background 0.28s ease;
}

.group-card:hover {
  transform: translateY(-6px);
  border-color: color-mix(in srgb, var(--n-primary-color) 42%, var(--n-border-color) 58%);
  box-shadow:
    0 20px 40px rgba(15, 23, 42, 0.12),
    0 8px 18px rgba(59, 130, 246, 0.08),
    inset 0 1px 0 rgba(255, 255, 255, 0.72);
}

.group-card__glow {
  position: absolute;
  top: -44px;
  right: -34px;
  width: 130px;
  height: 130px;
  border-radius: 9999px;
  background: color-mix(in srgb, var(--n-primary-color) 20%, transparent);
  filter: blur(18px);
  opacity: 0.9;
  pointer-events: none;
}

.group-card__grid {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.03) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.03) 1px, transparent 1px);
  background-size: 18px 18px;
  mask-image: linear-gradient(180deg, rgba(0, 0, 0, 0.28), transparent 72%);
  pointer-events: none;
}

.group-card__avatar,
.group-card__title,
.group-card__desc,
.group-card__footer {
  position: relative;
  z-index: 1;
}

.group-card__avatar {
  color: var(--n-primary-color);
  background:
    linear-gradient(135deg, color-mix(in srgb, var(--n-primary-color) 18%, white 82%), color-mix(in srgb, var(--n-primary-color) 6%, transparent));
  border: 1px solid color-mix(in srgb, var(--n-primary-color) 22%, transparent);
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.7),
    0 8px 16px rgba(59, 130, 246, 0.12);
}

.group-card__title {
  color: var(--n-text-color);
  letter-spacing: -0.01em;
}

.group-card__desc {
  color: var(--n-text-color-3);
  line-height: 1.65;
  min-height: 48px;
}

.group-card__footer {
  color: var(--n-text-color-3);
  border-top: 1px solid color-mix(in srgb, var(--n-border-color) 82%, transparent);
}

.group-card__action {
  color: var(--n-primary-color);
  font-weight: 600;
  transition: transform 0.28s ease;
}

.group-card:hover .group-card__action {
  transform: translateX(4px);
}
</style>

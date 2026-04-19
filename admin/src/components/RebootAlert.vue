<template>
	<n-modal
		:title="alertData.title"
		v-model:show="visible"
		:mask-closable="false"
		preset="card"
		style="width: 300px; background-color: white"
	>
		<div v-if="alertData.msg">{{ alertData.msg }}</div>
		<div class="text-sm text-gray-500">
			如果确认操作，请手动输入
			<span class="text-red-600">'{{ alertData.input }}'</span>
		</div>
		<div class="mt-4">
			<n-input v-model:value="inputVal" placeholder=""></n-input>
		</div>
		<template #footer>
			<div class="flex justify-end gap-4">
				<n-button @click="close">取消</n-button>
				<n-button type="primary" @click="confirm" :disabled="disabled" :loading="loading">确定</n-button>
			</div>
		</template>
	</n-modal>
</template>
<script setup lang="ts">
import { isSucc } from "@/utils/is"
import { MsgSuccess } from "@/utils/message"
const emit = defineEmits(["confirm"])
const visible = ref(false)
const alertData = ref<any>({})
const inputVal = ref("")
const loading = ref(false)
const disabled = computed(() => {
	if (!alertData.value.input) return false
	if (!inputVal.value) return true
	return alertData.value.input != inputVal.value
})
const open = (data: any) => {
	alertData.value = data
	inputVal.value = ""
	visible.value = true
}
const close = () => {
	visible.value = false
}
const confirm = () => {
	emit("confirm", loading)
	// loading.value = true
	// alertData.value.api().then((res:any)=>{
	//   if(isSucc(res.code)){
	//     MsgSuccess(res.msg)
	//     close()
	//   }
	// }).finally(()=>{
	//   loading.value = false
	// })
}
defineExpose({
	open,
	close
})
</script>

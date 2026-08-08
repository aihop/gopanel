<template>
	<slot />
</template>

<script setup lang="ts">
import { useGlobalActions } from "@/composables/useGlobalActions"
import { setGlobalMessageApi } from "@/utils/message"
import { useMessage, useNotification } from "naive-ui"

const message = useMessage()
const notification = useNotification()

setGlobalMessageApi(message)
useGlobalActions().init({ message, notification })
</script>

<template>
	<div>
		<RouterButton :buttons="buttons" />
		<LayoutContent>
			<router-view></router-view>
		</LayoutContent>
	</div>
</template>

<script lang="ts" setup>
import LayoutContent from "@/components/LayoutContent.vue"
import RouterButton from "@/components/RouterButton.vue"
import { t } from "@/i18n"

const buttons = [
	{
		label: t("container.container", 2),
		path: "/container/index"
	},
	{
		label: t("container.compose", 2),
		path: "/container/compose"
	},
	{
		label: t("container.image", 2),
		path: "/container/image"
	},
	{
		label: t("container.network", 2),
		path: "/container/network"
	},
	{
		label: t("container.volume", 2),
		path: "/container/volume"
	},
	{
		label: t("container.repo", 2),
		path: "/container/repo"
	},
	{
		label: t("container.composeTemplate", 2),
		path: "/container/template"
	},
	{
		label: t("container.setting", 2),
		path: "/container/setting"
	}
]
</script>

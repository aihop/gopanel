declare module "vuevectormap" {
	import type { Component, ComponentOptions, ComponentPublicInstance, Plugin } from "vue"

	export type VueVectorMapComponent = ComponentPublicInstance

	declare const VueVectorMap: Component & ComponentOptions<VueVectorMapComponent> & Plugin

	export default VueVectorMap
}

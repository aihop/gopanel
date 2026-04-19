import { setRouter } from "@/api"
import App from "@/App.vue"
import { i18n } from "@/i18n"
import formatExtensions from "@/plugin/format"
import i18nExtensions from "@/plugin/i18n"
import router from "@/router"
import { createPinia } from "pinia"
import { createPersistedState } from "pinia-plugin-persistedstate"

import { createApp } from "vue"

const meta = document.createElement("meta")
meta.name = "naive-ui-style"
document.head.appendChild(meta)

const pinia = createPinia()
pinia.use(
	createPersistedState({
		key: id => `__gopanel__${id}`,
		storage: localStorage
	})
)

const app = createApp(App)

app.use(pinia)
app.use(i18n)
app.use(i18nExtensions)
app.use(formatExtensions)
app.use(router)

setRouter(router)

app.mount("#app")

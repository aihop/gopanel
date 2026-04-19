import type { Locale, MessageSchema } from "./config"
import { createI18n } from "vue-i18n"
import { getI18NConf } from "./config"
import type { Composer } from "vue-i18n"

export const i18n = createI18n<MessageSchema, Locale>(getI18NConf() as any)

const i18nInstance = i18n.global as unknown as Composer
export const t = i18nInstance.t

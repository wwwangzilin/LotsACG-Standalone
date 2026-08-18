import 'default-passive-events'
import { StyleProvider } from '@varlet/ui'
import type { Pinia } from 'pinia'
import '@varlet/touch-emulator'

export default defineNuxtPlugin(() => {
  if (import.meta.client) {
    StyleProvider(usePiniaStore(usePinia() as Pinia).preferLight ? lightTheme : darkTheme)
  }
})

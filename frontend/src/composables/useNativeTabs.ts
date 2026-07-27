import { onBeforeUnmount, onMounted, watch, type WatchStopHandle } from 'vue'
import { usePlatformStore } from '@/stores/platform'
import { useUIStore } from '@/stores/ui'

const destinations = ['library', 'offline', 'search', 'settings'] as const

export function useNativeTabs() {
  const platformStore = usePlatformStore()
  const uiStore = useUIStore()
  let listening = false
  let stopWatching: WatchStopHandle | undefined

  const handleNativeTab = (event: Event) => {
    const index = Number((event as CustomEvent<{ index?: number }>).detail?.index)
    const destination = destinations[index]
    if (destination) uiStore.selectTopLevelDestination(destination)
  }

  const setListening = (enabled: boolean) => {
    if (enabled === listening) return
    listening = enabled
    if (enabled) {
      window.addEventListener('nativeTabSelected', handleNativeTab)
    } else {
      window.removeEventListener('nativeTabSelected', handleNativeTab)
    }
  }

  onMounted(() => {
    stopWatching = watch(
      () => platformStore.capabilities.nativeTopLevelTabs,
      setListening,
      { immediate: true },
    )
  })

  onBeforeUnmount(() => {
    stopWatching?.()
    setListening(false)
  })
}

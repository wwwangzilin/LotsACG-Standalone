export const useSmallWindow = () => {
  const { width } = useWindowSize()
  return computed(() => width.value <= 768)
}

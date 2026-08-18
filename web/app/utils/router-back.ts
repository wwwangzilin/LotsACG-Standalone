export const routerBack = () => {
  if (history.length > 1) {
    useRouter().back()
  } else {
    navigateTo('/')
  }
}

export const IsObjectIDValid = (id: string): boolean => {
  return /^[a-f\d]{24}$/i.test(id)
}

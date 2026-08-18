<template>
  <div>
    <var-paper ripple :elevation="elevation" v-hover="handleHover" class="tag-paper">
      <var-link underline="none" :to="`/tag/${encodeURIComponent(text)}`">
        <var-chip class="chip">{{ text }}</var-chip>
      </var-link>
    </var-paper>
  </div>
</template>

<script lang="ts" setup>
defineProps<{
  text: string
}>()

const elevation = ref(1)
const handleHover = (isHovering: boolean) => {
  elevation.value = isHovering ? 4 : 1
}
</script>

<style scoped>
.tag-paper {
  transition: box-shadow 0.1s ease-in-out;
  background-color: transparent;
  border: 2px solid rgba(0, 213, 255, 0.2);
  border-radius: 8px;
}

.chip {
  cursor: pointer;
  background-color: transparent;
  font-weight: bolder;
  transition: background-color 0.2s ease;
}

.chip:hover {
  background-color: rgba(0, 213, 255, 0.1);
}
</style>

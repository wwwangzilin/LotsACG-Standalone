<template>
  <NuxtLoadingIndicator :throttle="100" />
  <NuxtLayout>
    <NuxtPage keepalive />
  </NuxtLayout>
</template>

<script setup lang="ts">
useHead({
  meta: [
    {
      name: 'description',
      content: '我会带上我的萝莉私奔'
    },
    {
      name: 'application-name',
      content: 'ManyACG'
    }
  ],
  link: [
    {
      rel: 'icon',
      href: '/favicon.ico'
    }
  ],
  htmlAttrs: {
    lang: 'zh-CN'
  }
})

useSeoMeta({
  applicationName: 'ManyACG',
  ogDescription: '我会带上我的萝莉私奔',
  ogImage: '/logo.webp',
  ogSiteName: 'ManyACG',
  ogTitle: 'ManyACG',
  ogType: 'website',
  twitterCard: 'summary',
  twitterSite: '@ManyACG',
  twitterTitle: 'ManyACG',
  twitterDescription: '我会带上我的萝莉私奔',
  twitterImage: '/logo.webp'
})

// const cookie = useCookie('TOKEN')
// if (cookie.value && cookie.value !== 'undefined') {
//   const { data } = await useAcgapiData<ProfileResponse>('/user/profile', {
//     headers: {
//       Authorization: `Bearer ${cookie.value}`
//     }
//   })
//   if (data.value && data.value.data && data.value.data.settings) {
//     // ? wtf
//     usePiniaStore().setR18(data.value.data.settings.r18)
//   }
// }
</script>

<style>
::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

::-webkit-scrollbar-track {
  background: transparent;
  border-radius: 8px;
}

::-webkit-scrollbar-thumb {
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.35), rgba(96, 165, 250, 0.35));
  border-radius: 8px;
  transition: background-color 0.3s ease;
}

::-webkit-scrollbar-thumb:hover {
  background: linear-gradient(180deg, rgba(37, 99, 235, 0.7), rgba(96, 165, 250, 0.7));
}

::-webkit-scrollbar-corner {
  background: transparent;
}

body {
  /* 深空蓝黑/纯净浅底 + 极淡的蓝色氛围光 */
  background-color: hsla(var(--hsl-body), 0.99);
  background-image:
    radial-gradient(1200px 560px at 20% -12%, rgba(59, 130, 246, 0.07), transparent 60%),
    radial-gradient(900px 480px at 85% -8%, rgba(37, 99, 235, 0.05), transparent 60%),
    linear-gradient(180deg, hsla(var(--hsl-body), 0.99), hsla(var(--hsl-body), 0.99));
  background-size: cover;
  background-attachment: fixed;
  background-position: center;
  background-blend-mode: overlay;
  transition: background-color 0.2s ease, background-image 0.3s ease;
  color: var(--color-text);
  overflow-x: hidden;
  overflow-y: scroll;
  font-family: 'HarmonyOS Sans SC', 'Noto Sans SC', 'Noto Sans TC', system-ui, -apple-system, sans-serif;
  -webkit-font-smoothing: antialiased;
}

/* 选择色跟随主题 */
::selection {
  background: rgba(37, 99, 235, 0.3);
}

.kawaii-bounce-enter-active,
.kawaii-bounce-leave-active {
  transition: all 0.3s cubic-bezier(0.68, -1.2, 0.32, 2.2);
  transform-origin: center center;
  transform-style: preserve-3d;
}

.kawaii-bounce-enter-from {
  opacity: 0;
  transform: translateY(30px) rotateX(30deg) scale(0.92);
  filter: blur(2px);
}

.kawaii-bounce-enter-to {
  opacity: 1;
  transform: translateY(0) rotateX(0) scale(1);
  filter: blur(0);
}

.kawaii-bounce-leave-active {
  transition: all 0.2s cubic-bezier(0.645, 0.045, 0.355, 1);
}

.kawaii-bounce-leave-from {
  opacity: 1;
  transform: translateY(0) scale(1);
}

.kawaii-bounce-leave-to {
  opacity: 0;
  transform: translateY(-30px) scale(0.95);
  filter: blur(1.5px);
}

.fade-enter-active,
.fade-leave-active {
  transition: all 0.1s;
}

.fade-enter-from,
.fade-leave-to {
  opacity: 0;
  filter: blur(1rem);
}

/* 页面切换过渡 */
.page-fade-enter-active,
.page-fade-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.page-fade-enter-from {
  opacity: 0;
  transform: translateY(10px);
}

.page-fade-leave-to {
  opacity: 0;
  transform: translateY(-10px);
}
</style>

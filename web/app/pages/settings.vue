<template>
  <div class="settings">
    <h2>⚙️ 设置</h2>

    <var-card title="瀑布流设置" class="card">
      <var-cell title="每行最大列数 (宽屏)" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-slider v-model="local.maxColumnCount" :min="2" :max="10" :step="1" />
              <span class="value">{{ local.maxColumnCount }} 列</span>
            </div>
            <div class="desc">屏幕宽度足够时最多并排 {{ local.maxColumnCount }} 张卡片</div>
            <div class="cols-preview" :style="{ gridTemplateColumns: `repeat(${local.maxColumnCount}, 1fr)` }">
              <div v-for="i in local.maxColumnCount" :key="'max' + i" class="preview-cell"></div>
            </div>
          </div>
        </template>
      </var-cell>

      <var-cell title="每行最小列数 (窄屏)" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-slider v-model="local.minColumnCount" :min="1" :max="6" :step="1" />
              <span class="value">{{ local.minColumnCount }} 列</span>
            </div>
            <div class="desc">手机等窄屏下至少并排 {{ local.minColumnCount }} 张卡片</div>
            <div class="cols-preview" :style="{ gridTemplateColumns: `repeat(${local.minColumnCount}, 1fr)` }">
              <div v-for="i in local.minColumnCount" :key="'min' + i" class="preview-cell"></div>
            </div>
          </div>
        </template>
      </var-cell>

      <var-cell title="卡片最小宽度" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-slider v-model="local.itemMinWidth" :min="200" :max="600" :step="10" />
              <span class="value">{{ local.itemMinWidth }} px</span>
            </div>
            <div class="desc">每张卡片至少 {{ local.itemMinWidth }}px 宽，值越小同样宽度下列数越多</div>
          </div>
        </template>
      </var-cell>

      <var-cell title="预加载屏数" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-slider v-model="local.preloadScreenCount" :min="0" :max="3" :step="1" />
              <span class="value">{{ local.preloadScreenCount }} 屏</span>
            </div>
            <div class="desc" v-if="local.preloadScreenCount === 0">不提前加载，滚动到才加载（最省流量）</div>
            <div class="desc" v-else>
              滚动前提前加载 {{ local.preloadScreenCount }} 个屏幕高度的图片，值越大越流畅但更耗流量
            </div>
          </div>
        </template>
      </var-cell>

      <var-cell title="滚动加载触发距离" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-slider v-model="local.bottomDistance" :min="200" :max="2000" :step="50" />
              <span class="value">{{ local.bottomDistance }} px</span>
            </div>
            <div class="desc">距底部还有 {{ local.bottomDistance }}px 时就开始加载下一页</div>
          </div>
        </template>
      </var-cell>

      <var-cell title="仅图片模式" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-switch v-model="local.onlyImage" />
              <span class="switch-state">{{ local.onlyImage ? '已开启' : '已关闭' }}</span>
            </div>
            <div class="desc">开启后卡片不显示标题和作者，滚动更流畅</div>
          </div>
        </template>
      </var-cell>
    </var-card>

    <var-card title="内容设置" class="card">
      <var-cell title="R18 内容" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-switch v-model="local.r18" />
              <span class="switch-state">{{ local.r18 ? '已开启' : '已关闭' }}</span>
            </div>
            <div class="desc">{{ local.r18 ? '将显示 R18 内容' : '不显示 R18 内容' }}</div>
          </div>
        </template>
      </var-cell>
      <var-cell title="浅色主题" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="cell-control">
              <var-switch v-model="local.preferLight" />
              <span class="switch-state">{{ local.preferLight ? '浅色' : '深色' }}</span>
            </div>
            <div class="desc">{{ local.preferLight ? '使用明亮主题' : '使用深色主题' }}</div>
          </div>
        </template>
      </var-cell>
    </var-card>

    <div class="actions">
      <var-button type="primary" @click="save">保存并刷新</var-button>
      <var-button @click="reset">恢复默认</var-button>
    </div>

    <var-card title="Bot 操控" class="card">
      <var-cell title="Bot 状态" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="status-line" v-if="status">
              <span class="dot" :class="{ ok: status.running }"></span>
              <span>@{{ status.bot_username || '-' }}</span>
              <span v-if="status.channel_name">· 主频道 {{ status.channel_name }}</span>
              <span v-if="status.group_id">· 群 {{ status.group_id }}</span>
              <span>{{ status.running ? '运行中' : '未运行' }}</span>
            </div>
            <div class="desc" v-if="!status && !statusLoading">Bot 未启用或未配置</div>
            <div class="desc" v-if="statusLoading">加载中...</div>
          </div>
        </template>
      </var-cell>
      <var-cell title="推送到主频道" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="bot-row">
              <var-input v-model="postUrl" placeholder="作品链接 (pixiv / twitter / ...)" class="flex-1" />
              <var-button type="primary" size="small" :loading="posting" @click="postToChannel">推送</var-button>
            </div>
            <div class="desc">将作品发布到配置的主频道 {{ status?.channel_name || '' }}</div>
          </div>
        </template>
      </var-cell>
      <var-cell title="发送到指定群" :border="false">
        <template #default>
          <div class="setting-block">
            <div class="bot-col">
              <var-input v-model="sendUrl" placeholder="作品链接" />
              <div class="bot-row">
                <var-input v-model="chatId" placeholder="群/用户 Chat ID" class="flex-1" />
                <var-button type="primary" size="small" :loading="sending" @click="sendInfo">发送</var-button>
              </div>
            </div>
            <div class="desc">把作品信息发送到任意群或用户 (Chat ID 可从 bot 处获取)</div>
          </div>
        </template>
      </var-cell>
    </var-card>
  </div>
</template>

<script lang="ts" setup>
import { Snackbar, StyleProvider } from '@varlet/ui'
import type { BaseResponse } from '~/types/artwork'

useHead({
  title: '设置'
})

const piniaStore = usePiniaStore()

const local = reactive({
  ...piniaStore.waterfall,
  r18: piniaStore.r18,
  preferLight: piniaStore.preferLight
})

const save = () => {
  piniaStore.setWaterfall({
    itemMinWidth: local.itemMinWidth,
    minColumnCount: local.minColumnCount,
    maxColumnCount: local.maxColumnCount,
    preloadScreenCount: local.preloadScreenCount,
    bottomDistance: local.bottomDistance,
    onlyImage: local.onlyImage
  })
  piniaStore.setR18(local.r18)
  piniaStore.setpreferLight(local.preferLight)
  StyleProvider(local.preferLight ? lightTheme : darkTheme)
  Snackbar.success('已保存')
  if (import.meta.client) {
    window.location.reload()
  }
}

const reset = () => {
  piniaStore.resetWaterfall()
  local.itemMinWidth = piniaStore.waterfall.itemMinWidth
  local.minColumnCount = piniaStore.waterfall.minColumnCount
  local.maxColumnCount = piniaStore.waterfall.maxColumnCount
  local.preloadScreenCount = piniaStore.waterfall.preloadScreenCount
  local.bottomDistance = piniaStore.waterfall.bottomDistance
  local.onlyImage = piniaStore.waterfall.onlyImage
  Snackbar.info('已恢复默认, 点击「保存并刷新」生效')
}

// ---------- Bot 操控 (无缝, 直接使用已配置的 bot, 无需 API Key) ----------
const statusLoading = ref(false)
const status = ref<any>(null)
const loadStatus = async () => {
  statusLoading.value = true
  try {
    const resp = await $acgapi<BaseResponse<any>>('/bot/status', { method: 'GET' })
    status.value = resp.data
  } catch (e: any) {
    status.value = null
  } finally {
    statusLoading.value = false
  }
}

onMounted(() => {
  loadStatus()
})

const posting = ref(false)
const postUrl = ref('')
const postToChannel = async () => {
  if (!postUrl.value) {
    Snackbar.warning('请填写作品链接')
    return
  }
  posting.value = true
  try {
    await $acgapi<BaseResponse<string>>('/bot/post_artwork', {
      method: 'POST',
      body: { source_url: postUrl.value }
    })
    Snackbar.success('已推送到主频道')
  } catch (e: any) {
    Snackbar.error('推送失败: ' + (e?.data?.message || e?.message || '未知错误'))
  } finally {
    posting.value = false
  }
}

const sending = ref(false)
const sendUrl = ref('')
const chatId = ref('')
const sendInfo = async () => {
  if (!sendUrl.value) {
    Snackbar.warning('请填写作品链接')
    return
  }
  const id = Number(chatId.value)
  if (!id) {
    Snackbar.warning('请填写有效的 Chat ID')
    return
  }
  sending.value = true
  try {
    await $acgapi<BaseResponse<string>>('/bot/send_artwork_info', {
      method: 'POST',
      body: { source_url: sendUrl.value, chat_id: id }
    })
    Snackbar.success('已发送')
  } catch (e: any) {
    Snackbar.error('发送失败: ' + (e?.data?.message || e?.message || '未知错误'))
  } finally {
    sending.value = false
  }
}
</script>

<style scoped>
.settings {
  margin: 0 auto;
  padding: 20px;
  max-width: 900px;
}

.card {
  margin-bottom: 20px;
}

.setting-block {
  display: flex;
  flex-direction: column;
  gap: 6px;
  width: 100%;
  padding: 4px 0;
}

.cell-control {
  display: flex;
  align-items: center;
  gap: 12px;
}

.cell-control .var-slider {
  flex: 1;
}

.value {
  min-width: 56px;
  text-align: right;
  font-variant-numeric: tabular-nums;
  font-weight: 600;
}

.switch-state {
  min-width: 48px;
  font-variant-numeric: tabular-nums;
}

.desc {
  font-size: 12px;
  color: rgba(128, 128, 128, 0.9);
  line-height: 1.4;
}

.cols-preview {
  display: grid;
  gap: 4px;
  margin-top: 2px;
}

.preview-cell {
  height: 16px;
  border-radius: 4px;
  background: linear-gradient(135deg, rgba(68, 194, 219, 0.35), rgba(68, 194, 219, 0.15));
}

.actions {
  display: flex;
  gap: 12px;
  margin-bottom: 20px;
}

.bot-row {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.bot-col {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}

.flex-1 {
  flex: 1;
}

.status-line {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  font-weight: 500;
}

.dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: #f44336;
}

.dot.ok {
  background: #4caf50;
}
</style>

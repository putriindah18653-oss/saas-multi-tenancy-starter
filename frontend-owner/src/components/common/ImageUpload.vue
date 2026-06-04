<template>
  <div
    class="image-upload"
    :class="{ 'image-upload--dragover': dragover }"
    @dragover.prevent="dragover = true"
    @dragleave.prevent="dragover = false"
    @drop.prevent="onDrop"
  >
    <div class="image-upload__preview" :class="shape === 'circle' ? 'image-upload__preview--circle' : ''">
      <img v-if="previewSrc && !previewFailed" :src="previewSrc" :alt="alt" @error="previewFailed = true" @load="previewFailed = false" />
      <div v-else class="image-upload__placeholder">
        <span class="image-upload__initials">{{ fallbackInitials }}</span>
      </div>
    </div>

    <div class="image-upload__body">
      <input ref="fileInput" type="file" class="hidden" :accept="accept" aria-label="Choose image file" @change="onFileSelect" />

      <p class="image-upload__title">{{ title }}</p>
      <p class="image-upload__hint">{{ hint }}</p>

      <div class="image-upload__actions">
        <button type="button" class="image-upload__btn image-upload__btn--primary" @click="fileInput?.click()">
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            <polyline points="17 8 12 3 7 8" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            <line x1="12" y1="3" x2="12" y2="15" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          {{ pendingFile ? 'Change' : 'Choose image' }}
        </button>
        <button v-if="previewSrc" type="button" class="image-upload__btn image-upload__btn--danger" @click="clear">
          <svg class="h-4 w-4" viewBox="0 0 24 24" fill="none" aria-hidden="true">
            <polyline points="3 6 5 6 21 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
            <path d="M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          Remove
        </button>
      </div>

      <p v-if="pendingFile" class="image-upload__pending">{{ pendingFile.name }} — akan diupload saat save.</p>
      <p v-if="fileError" class="image-upload__error" role="alert">{{ fileError }}</p>
      <p class="image-upload__meta">{{ meta }}</p>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  modelValue: string
  title?: string
  hint?: string
  meta?: string
  alt?: string
  accept?: string
  shape?: 'rounded' | 'circle'
  initials?: string
  resolveUrl?: (url: string) => string
}>(), {
  title: 'Upload image',
  hint: 'Drag & drop atau klik upload',
  meta: 'PNG, JPG, WebP, AVIF. Max 8 MB.',
  alt: 'Preview',
  accept: 'image/png,image/jpeg,image/webp,image/avif',
  shape: 'rounded',
  initials: '',
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
  'update:file': [file: File | null]
}>()

const allowedImageTypes = new Set(['image/jpeg', 'image/png', 'image/webp', 'image/avif'])
const maxImageSize = 8 * 1024 * 1024

const fileInput = ref<HTMLInputElement | null>(null)
const dragover = ref(false)
const previewFailed = ref(false)
const pendingFile = ref<File | null>(null)
const localPreview = ref<string | null>(null)
const fileError = ref('')

const resolvedUrl = computed(() => {
  if (!props.modelValue) return ''
  return props.resolveUrl ? props.resolveUrl(props.modelValue) : props.modelValue
})

const previewSrc = computed(() => localPreview.value || resolvedUrl.value)
const fallbackInitials = computed(() => props.initials || '?')

function setFile(file: File | null) {
  if (localPreview.value) URL.revokeObjectURL(localPreview.value)
  if (file) {
    if (!isValidImage(file)) {
      localPreview.value = null
      pendingFile.value = null
      previewFailed.value = false
      emit('update:file', null)
      return
    }

    localPreview.value = URL.createObjectURL(file)
    previewFailed.value = false
  } else {
    localPreview.value = null
    fileError.value = ''
  }
  pendingFile.value = file
  emit('update:file', file)
}

function isValidImage(file: File) {
  if (!allowedImageTypes.has(file.type)) {
    fileError.value = 'Format gambar harus JPG, PNG, WebP, atau AVIF.'
    return false
  }

  if (file.size > maxImageSize) {
    fileError.value = 'Ukuran gambar maksimal 8 MB.'
    return false
  }

  fileError.value = ''
  return true
}

function onFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (file) setFile(file)
  input.value = ''
}

function onDrop(event: DragEvent) {
  dragover.value = false
  const file = event.dataTransfer?.files?.[0]
  if (file) setFile(file)
}

function clear() {
  setFile(null)
  emit('update:modelValue', '')
}

watch(() => props.modelValue, (val) => {
  if (val && !pendingFile.value) {
    // External value changed (e.g. after save), clear local preview
    if (localPreview.value) {
      URL.revokeObjectURL(localPreview.value)
      localPreview.value = null
    }
    pendingFile.value = null
  }
})

onBeforeUnmount(() => {
  if (localPreview.value) URL.revokeObjectURL(localPreview.value)
})

defineExpose({ pendingFile })
</script>

<style scoped>
.image-upload {
  display: grid;
  grid-template-columns: auto 1fr;
  gap: 1.25rem;
  align-items: center;
  padding: 1.25rem;
  border-radius: var(--radius-card);
  border: 1.5px dashed var(--border-strong);
  background: var(--surface);
  transition: border-color 0.2s, background 0.2s;
}

.image-upload--dragover {
  border-color: var(--accent);
  background: rgba(53, 211, 153, 0.04);
}

.image-upload__preview {
  position: relative;
  width: 88px;
  height: 88px;
  border-radius: var(--radius-card);
  overflow: hidden;
  border: 1.5px solid var(--border-strong);
  background: linear-gradient(135deg, rgba(255,255,255,0.06), rgba(255,255,255,0.02));
  flex-shrink: 0;
}

.image-upload__preview--circle {
  border-radius: 50%;
}

.image-upload__preview img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.image-upload__placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 100%;
  height: 100%;
}

.image-upload__initials {
  font-size: 1.25rem;
  font-weight: 700;
  color: var(--accent);
}

.image-upload__body {
  min-width: 0;
}

.image-upload__title {
  font-size: 0.875rem;
  font-weight: 600;
  color: var(--text-primary);
}

.image-upload__hint {
  margin-top: 0.2rem;
  font-size: 0.75rem;
  color: var(--text-muted);
}

.image-upload__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 0.5rem;
  margin-top: 0.75rem;
}

.image-upload__btn {
  display: inline-flex;
  align-items: center;
  gap: 0.4rem;
  padding: 0.4rem 0.75rem;
  font-size: 0.8rem;
  font-weight: 600;
  border-radius: var(--radius-button);
  border: 1px solid transparent;
  transition: background 0.15s, border-color 0.15s;
  cursor: pointer;
}

.image-upload__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.image-upload__btn--primary {
  background: var(--accent);
  color: white;
}

.image-upload__btn--primary:hover:not(:disabled) {
  background: var(--accent-hover);
}

.image-upload__btn--danger {
  border-color: rgba(255, 100, 103, 0.3);
  background: rgba(255, 100, 103, 0.08);
  color: var(--danger);
}

.image-upload__btn--danger:hover:not(:disabled) {
  background: rgba(255, 100, 103, 0.16);
}

.image-upload__pending {
  margin-top: 0.5rem;
  font-size: 0.75rem;
  font-weight: 500;
  color: var(--accent);
}

.image-upload__error {
  margin-top: 0.5rem;
  font-size: 0.75rem;
  font-weight: 600;
  color: var(--danger);
}

.image-upload__meta {
  margin-top: 0.4rem;
  font-size: 0.7rem;
  color: var(--text-muted);
}

@media (max-width: 480px) {
  .image-upload {
    grid-template-columns: 1fr;
    justify-items: center;
    text-align: center;
  }

  .image-upload__actions {
    justify-content: center;
  }
}
</style>

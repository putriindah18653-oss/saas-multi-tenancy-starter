import { mount } from '@vue/test-utils'
import ImageUpload from './ImageUpload.vue'

describe('ImageUpload', () => {
  const createObjectURL = vi.fn(() => 'blob:preview')
  const revokeObjectURL = vi.fn()

  beforeEach(() => {
    vi.stubGlobal('URL', { createObjectURL, revokeObjectURL })
    createObjectURL.mockClear()
    revokeObjectURL.mockClear()
  })

  afterEach(() => {
    vi.unstubAllGlobals()
  })

  it('rejects non-image files before creating preview', async () => {
    const wrapper = mount(ImageUpload, { props: { modelValue: '' } })
    const file = new File(['x'], 'notes.txt', { type: 'text/plain' })
    const input = wrapper.find('input[type="file"]').element as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [file] })

    await wrapper.find('input[type="file"]').trigger('change')

    expect(createObjectURL).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:file')?.[0]).toEqual([null])
    expect(wrapper.text()).toContain('Format gambar harus JPG, PNG, WebP, atau AVIF.')
  })

  it('rejects images over 8 MB before creating preview', async () => {
    const wrapper = mount(ImageUpload, { props: { modelValue: '' } })
    const file = new File(['x'], 'large.png', { type: 'image/png' })
    Object.defineProperty(file, 'size', { value: 8 * 1024 * 1024 + 1 })
    const input = wrapper.find('input[type="file"]').element as HTMLInputElement
    Object.defineProperty(input, 'files', { value: [file] })

    await wrapper.find('input[type="file"]').trigger('change')

    expect(createObjectURL).not.toHaveBeenCalled()
    expect(wrapper.emitted('update:file')?.[0]).toEqual([null])
    expect(wrapper.text()).toContain('Ukuran gambar maksimal 8 MB.')
  })
})
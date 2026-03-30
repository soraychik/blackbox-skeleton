import { describe, it, expect } from 'vitest'
import { getTheme } from '../theme'

describe('getTheme', () => {
  describe('light mode', () => {
    it('устанавливает palette.mode в light', () => {
      const theme = getTheme('light')
      expect(theme.palette.mode).toBe('light')
    })

    it('устанавливает правильные primary цвета', () => {
      const theme = getTheme('light')
      expect(theme.palette.primary.main).toBe('#2563eb')
      expect(theme.palette.primary.light).toBe('#60a5fa')
      expect(theme.palette.primary.dark).toBe('#1d4ed8')
    })

    it('устанавливает правильные background цвета', () => {
      const theme = getTheme('light')
      expect(theme.palette.background.default).toBe('#f8fafc')
      expect(theme.palette.background.paper).toBe('#ffffff')
    })

    it('устанавливает правильные text цвета', () => {
      const theme = getTheme('light')
      expect(theme.palette.text.primary).toBe('#1e293b')
      expect(theme.palette.text.secondary).toBe('#64748b')
    })

    it('устанавливает MuiTableCell head с правильным background для light mode', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiTableCell.styleOverrides.head.backgroundColor).toBe('#f1f5f9')
    })
  })

  describe('dark mode', () => {
    it('устанавливает palette.mode в dark', () => {
      const theme = getTheme('dark')
      expect(theme.palette.mode).toBe('dark')
    })

    it('устанавливает правильные primary цвета', () => {
      const theme = getTheme('dark')
      expect(theme.palette.primary.main).toBe('#3b82f6')
      expect(theme.palette.primary.light).toBe('#60a5fa')
      expect(theme.palette.primary.dark).toBe('#2563eb')
    })

    it('устанавливает правильные background цвета', () => {
      const theme = getTheme('dark')
      expect(theme.palette.background.default).toBe('#0f172a')
      expect(theme.palette.background.paper).toBe('#1e293b')
    })

    it('устанавливает правильные text цвета', () => {
      const theme = getTheme('dark')
      expect(theme.palette.text.primary).toBe('#f1f5f9')
      expect(theme.palette.text.secondary).toBe('#94a3b8')
    })

    it('устанавливает MuiTableCell head с правильным background для dark mode', () => {
      const theme = getTheme('dark')
      expect(theme.components.MuiTableCell.styleOverrides.head.backgroundColor).toBe('#1e293b')
    })
  })

  describe('typography', () => {
    it('устанавливает правильный fontFamily', () => {
      const theme = getTheme('light')
      expect(theme.typography.fontFamily).toContain('Inter')
    })

    it('устанавливает правильные стили для h1', () => {
      const theme = getTheme('light')
      expect(theme.typography.h1.fontSize).toBe('2rem')
      expect(theme.typography.h1.fontWeight).toBe(600)
    })

    it('устанавливает правильные стили для h2', () => {
      const theme = getTheme('light')
      expect(theme.typography.h2.fontSize).toBe('1.5rem')
      expect(theme.typography.h2.fontWeight).toBe(600)
    })

    it('устанавливает правильные стили для h3', () => {
      const theme = getTheme('light')
      expect(theme.typography.h3.fontSize).toBe('1.25rem')
      expect(theme.typography.h3.fontWeight).toBe(600)
    })

    it('устанавливает правильные стили для h4', () => {
      const theme = getTheme('light')
      expect(theme.typography.h4.fontSize).toBe('1.125rem')
      expect(theme.typography.h4.fontWeight).toBe(600)
    })
  })

  describe('shape', () => {
    it('устанавливает borderRadius в 8', () => {
      const theme = getTheme('light')
      expect(theme.shape.borderRadius).toBe(8)
    })
  })

  describe('MuiButton overrides', () => {
    it('убирает uppercase трансформацию текста', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiButton.styleOverrides.root.textTransform).toBe('none')
    })

    it('устанавливает fontWeight в 500', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiButton.styleOverrides.root.fontWeight).toBe(500)
    })
  })

  describe('MuiCard overrides', () => {
    it('устанавливает boxShadow', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiCard.styleOverrides.root.boxShadow).toBeTruthy()
    })
  })

  describe('MuiPaper overrides', () => {
    it('убирает backgroundImage', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiPaper.styleOverrides.root.backgroundImage).toBe('none')
    })
  })

  describe('MuiTableCell overrides', () => {
    it('устанавливает fontWeight в 600 для header ячеек', () => {
      const theme = getTheme('light')
      expect(theme.components.MuiTableCell.styleOverrides.head.fontWeight).toBe(600)
    })
  })
})

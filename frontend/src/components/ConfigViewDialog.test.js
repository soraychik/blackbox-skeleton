import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { ThemeProvider, createTheme } from '@mui/material'
import ConfigViewDialog from './ConfigViewDialog'

vi.mock('react-window', () => ({
  FixedSizeList: vi.fn(() => (
    <div data-testid="fixed-size-list">
      <span data-testid="list-placeholder" />
    </div>
  )),
}))

const renderWithTheme = (component, mode = 'light') => {
  const theme = createTheme({ palette: { mode } })
  return render(<ThemeProvider theme={theme}>{component}</ThemeProvider>)
}

const defaultProps = {
  open: true,
  onClose: vi.fn(),
}

describe('ConfigViewDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  describe('открытие и закрытие', () => {
    it('открывается с open=true', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} />)
      expect(screen.getByRole('dialog')).toBeInTheDocument()
    })

    it('закрывается по клику на крестик', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} />)
      const closeButton = screen.getByTestId('CloseIcon').closest('button')
      fireEvent.click(closeButton)
      expect(defaultProps.onClose).toHaveBeenCalledTimes(1)
    })

    it('не рендерит контент при open=false', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} open={false} />)
      expect(screen.queryByRole('dialog')).not.toBeInTheDocument()
    })
  })

  describe('title', () => {
    it('отображает переданный title', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} title="Тестовый заголовок" />)
      expect(screen.getByText('Тестовый заголовок')).toBeInTheDocument()
    })

    it('использует дефолтный title если не передан', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} />)
      expect(screen.getByText('Просмотр конфигурации')).toBeInTheDocument()
    })
  })

  describe('режим snippets', () => {
    const snippetsProps = {
      ...defaultProps,
      snippets: [
        { line: 1, text: 'config line 1', match: false },
        { line: 2, text: 'config line 2', match: true },
        { line: 3, text: 'config line 3', match: false },
      ],
    }

    it('отображает список сниппетов', () => {
      renderWithTheme(<ConfigViewDialog {...snippetsProps} />)
      expect(screen.getByText('config line 1')).toBeInTheDocument()
      expect(screen.getByText('config line 2')).toBeInTheDocument()
      expect(screen.getByText('config line 3')).toBeInTheDocument()
    })

    it('отображает номера строк', () => {
      renderWithTheme(<ConfigViewDialog {...snippetsProps} />)
      expect(screen.getByText('1')).toBeInTheDocument()
      expect(screen.getByText('2')).toBeInTheDocument()
      expect(screen.getByText('3')).toBeInTheDocument()
    })

    it('показывает "Нет сниппетов" для пустого массива', () => {
      renderWithTheme(<ConfigViewDialog {...defaultProps} snippets={[]} />)
      expect(screen.getByText('Нет сниппетов')).toBeInTheDocument()
    })

    it('не показывает snippets когда есть контент', () => {
      renderWithTheme(
        <ConfigViewDialog
          {...defaultProps}
          content="some content"
          snippets={[{ line: 1, text: 'snippet', match: false }]}
        />
      )
      expect(screen.queryByText('snippet')).not.toBeInTheDocument()
    })
  })

  describe('режим полного контента (full file mode)', () => {
    const contentProps = {
      ...defaultProps,
      content: 'line 1\nline 2\nline 3',
    }

    it('рендерит FixedSizeList для контента', () => {
      renderWithTheme(<ConfigViewDialog {...contentProps} />)
      expect(screen.getByTestId('fixed-size-list')).toBeInTheDocument()
    })

    it('не показывает кнопку "Посмотреть весь файл" когда передан content', () => {
      renderWithTheme(
        <ConfigViewDialog
          {...defaultProps}
          content="some content"
          snippets={[{ line: 1, text: 'snippet', match: false }]}
          onViewFullFile={vi.fn()}
        />
      )
      expect(screen.queryByText('Посмотреть весь файл')).not.toBeInTheDocument()
    })
  })

  describe('кнопка "Посмотреть весь файл"', () => {
    it('отображается только когда snippets !== null и контент не загружен', () => {
      renderWithTheme(
        <ConfigViewDialog
          {...defaultProps}
          snippets={[{ line: 1, text: 'config', match: false }]}
          onViewFullFile={vi.fn()}
        />
      )
      expect(screen.getByText('Посмотреть весь файл')).toBeInTheDocument()
    })

    it('вызывает onViewFullFile по клику', () => {
      const onViewFullFile = vi.fn().mockResolvedValue('full content')
      renderWithTheme(
        <ConfigViewDialog
          {...defaultProps}
          snippets={[{ line: 1, text: 'config', match: false }]}
          onViewFullFile={onViewFullFile}
        />
      )
      fireEvent.click(screen.getByText('Посмотреть весь файл'))
      expect(onViewFullFile).toHaveBeenCalledTimes(1)
    })

    it('disabled во время загрузки', () => {
      const onViewFullFile = vi.fn().mockImplementation(() => new Promise(() => {}))
      renderWithTheme(
        <ConfigViewDialog
          {...defaultProps}
          snippets={[{ line: 1, text: 'config', match: false }]}
          onViewFullFile={onViewFullFile}
        />
      )
      fireEvent.click(screen.getByText('Посмотреть весь файл'))
      expect(screen.getByText('Посмотреть весь файл').closest('button')).toBeDisabled()
    })
  })

  describe('кнопка "Скачать"', () => {
    const downloadProps = {
      ...defaultProps,
      content: 'config content',
      onDownload: vi.fn(),
    }

    it('отображается только когда передан onDownload', () => {
      renderWithTheme(<ConfigViewDialog {...downloadProps} />)
      expect(screen.getByText('Скачать')).toBeInTheDocument()
    })

    it('не отображается когда onDownload не передан', () => {
      renderWithTheme(
        <ConfigViewDialog {...defaultProps} content="config" onDownload={undefined} />
      )
      expect(screen.queryByText('Скачать')).not.toBeInTheDocument()
    })

    it('вызывает onDownload по клику', () => {
      renderWithTheme(<ConfigViewDialog {...downloadProps} />)
      fireEvent.click(screen.getByText('Скачать'))
      expect(downloadProps.onDownload).toHaveBeenCalledTimes(1)
    })

    it('disabled при downloadLoading=true', () => {
      renderWithTheme(
        <ConfigViewDialog {...downloadProps} downloadLoading={true} />
      )
      expect(screen.getByText('Скачать').closest('button')).toBeDisabled()
    })
  })
})

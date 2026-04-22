import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { render, screen } from '@testing-library/react'
import { MemoryRouter, Routes, Route } from 'react-router-dom'
import ProtectedRoute from './ProtectedRoute'

const localStorageMock = (() => {
  let store = {}
  return {
    getItem: (key) => store[key] || null,
    setItem: (key, value) => { store[key] = value },
    removeItem: (key) => { delete store[key] },
    clear: () => { store = {} },
  }
})()
Object.defineProperty(window, 'localStorage', { value: localStorageMock })

const mockNavigate = vi.fn()

vi.mock('react-router-dom', async () => {
  const actual = await vi.importActual('react-router-dom')
  return {
    ...actual,
    Navigate: ({ to, replace }) => {
      mockNavigate({ to, replace })
      return null
    },
  }
})

const TestChild = () => <div data-testid="protected-content">Protected Content</div>
const LoginPage = () => <div data-testid="login-page">Login Page</div>

const renderWithRouter = (initialEntry = '/') => {
  render(
    <MemoryRouter initialEntries={[initialEntry]}>
      <Routes>
        <Route path="/login" element={<LoginPage />} />
        <Route element={<ProtectedRoute />}>
          <Route path="/" element={<TestChild />} />
          <Route path="/dashboard" element={<TestChild />} />
        </Route>
      </Routes>
    </MemoryRouter>
  )
}

describe('ProtectedRoute', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    localStorage.clear()
  })

  afterEach(() => {
    localStorage.clear()
  })

  describe('когда токен отсутствует', () => {
    it('редиректит на /login', () => {
      renderWithRouter('/')

      expect(mockNavigate).toHaveBeenCalledWith({ to: '/login', replace: true })
    })

    it('не рендерит защищённый контент', () => {
      renderWithRouter('/')

      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument()
    })

    it('редиректит с любого защищённого маршрута', () => {
      renderWithRouter('/dashboard')

      expect(mockNavigate).toHaveBeenCalledWith({ to: '/login', replace: true })
      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument()
    })
  })

  describe('когда токен присутствует', () => {
    it('рендерит защищённый контент', () => {
      localStorage.setItem('token', 'mock-token-123')

      renderWithRouter('/')

      expect(screen.queryByTestId('protected-content')).toBeInTheDocument()
    })

    it('не вызывает редирект', () => {
      localStorage.setItem('token', 'mock-token-123')

      renderWithRouter('/')

      expect(mockNavigate).not.toHaveBeenCalled()
    })

    it('работает с разными форматами токена', () => {
      localStorage.setItem('token', 'Bearer abc123')

      renderWithRouter('/')

      expect(screen.queryByTestId('protected-content')).toBeInTheDocument()
      expect(mockNavigate).not.toHaveBeenCalled()
    })
  })

  describe('edge cases', () => {
    it('пустая строка как токен считается отсутствием токена', () => {
      localStorage.setItem('token', '')

      renderWithRouter('/')

      expect(mockNavigate).toHaveBeenCalled()
      expect(screen.queryByTestId('protected-content')).not.toBeInTheDocument()
    })

    it('токен с пробелами считается валидным', () => {
      localStorage.setItem('token', '  valid-token  ')

      renderWithRouter('/')

      expect(screen.queryByTestId('protected-content')).toBeInTheDocument()
    })
  })
})

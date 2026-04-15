import '@testing-library/jest-dom'

class ResizeObserverMock {
  observe() {}
  disconnect() {}
  unobserve() {}
}
global.ResizeObserver = ResizeObserverMock
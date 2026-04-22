import { describe, it, expect, vi, beforeEach } from 'vitest'

vi.mock('./api', () => ({
  getDevices: vi.fn(() => Promise.resolve([])),
  getVersions: vi.fn(() => Promise.resolve([])),
  getDashboardStats: vi.fn(() => Promise.resolve({})),
  getVersionContent: vi.fn(() => Promise.resolve('')),
  getVersionDiff: vi.fn(() => Promise.resolve({})),
  searchPattern: vi.fn(() => Promise.resolve([])),
  getDeviceVersions: vi.fn(() => Promise.resolve({})),
  getLatestVersionsForDevices: vi.fn(() => Promise.resolve({})),
  getDevicesDiff: vi.fn(() => Promise.resolve({})),
  getDiffByDate: vi.fn(() => Promise.resolve({})),
  exportConfigByDate: vi.fn(() => Promise.resolve({})),
  searchChanges: vi.fn(() => Promise.resolve({})),
  triggerScan: vi.fn(() => Promise.resolve({})),
  getScanStatus: vi.fn(() => Promise.resolve({})),
  default: {
    get: vi.fn(),
    post: vi.fn(),
    interceptors: {
      response: {
        use: vi.fn(),
      },
    },
  },
}))

import {
  getDevices,
  getVersions,
  getDashboardStats,
  getVersionContent,
  getVersionDiff,
  searchPattern,
  getDeviceVersions,
  getLatestVersionsForDevices,
  getDevicesDiff,
  getDiffByDate,
  exportConfigByDate,
  searchChanges,
  triggerScan,
  getScanStatus,
} from './api'

describe('API functions', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getDevices.mockImplementation(() => Promise.resolve([]))
    getVersions.mockImplementation(() => Promise.resolve([]))
    getDashboardStats.mockImplementation(() => Promise.resolve({}))
    getVersionContent.mockImplementation(() => Promise.resolve(''))
    getVersionDiff.mockImplementation(() => Promise.resolve({}))
    searchPattern.mockImplementation(() => Promise.resolve([]))
    getDeviceVersions.mockImplementation(() => Promise.resolve({}))
    getLatestVersionsForDevices.mockImplementation(() => Promise.resolve({}))
    getDevicesDiff.mockImplementation(() => Promise.resolve({}))
    getDiffByDate.mockImplementation(() => Promise.resolve({}))
    exportConfigByDate.mockImplementation(() => Promise.resolve({}))
    searchChanges.mockImplementation(() => Promise.resolve({}))
    triggerScan.mockImplementation(() => Promise.resolve({}))
    getScanStatus.mockImplementation(() => Promise.resolve({}))
  })

  describe('getDevices', () => {
    it('возвращает массив устройств', async () => {
      const mockDevices = [{ id: 1, name: 'Device 1' }]
      getDevices.mockResolvedValue(mockDevices)

      const result = await getDevices()

      expect(result).toEqual(mockDevices)
    })

    it('возвращает пустой массив если devices отсутствует', async () => {
      getDevices.mockResolvedValue([])

      const result = await getDevices()

      expect(result).toEqual([])
    })
  })

  describe('getVersions', () => {
    it('возвращает массив версий', async () => {
      const mockVersions = [{ id: 1, version: '1.0.0' }]
      getVersions.mockResolvedValue(mockVersions)

      const result = await getVersions()

      expect(result).toEqual(mockVersions)
    })
  })

  describe('getDashboardStats', () => {
    it('возвращает данные статистики', async () => {
      const mockStats = { total: 10, active: 5 }
      getDashboardStats.mockResolvedValue(mockStats)

      const result = await getDashboardStats()

      expect(result).toEqual(mockStats)
    })
  })

  describe('getVersionContent', () => {
    it('возвращает контент версии', async () => {
      const mockContent = 'config content'
      getVersionContent.mockResolvedValue(mockContent)

      const result = await getVersionContent('123')

      expect(result).toEqual(mockContent)
    })
  })

  describe('getVersionDiff', () => {
    it('возвращает diff двух версий', async () => {
      const mockDiff = { changes: [] }
      getVersionDiff.mockResolvedValue(mockDiff)

      const result = await getVersionDiff('v1', 'v2')

      expect(result).toEqual(mockDiff)
    })
  })

  describe('searchPattern', () => {
    it('возвращает результаты поиска', async () => {
      const mockResults = [{ id: 1 }]
      searchPattern.mockResolvedValue(mockResults)

      const result = await searchPattern({ pattern: 'test' })

      expect(result).toEqual(mockResults)
    })

    it('возвращает пустой массив если results отсутствует', async () => {
      searchPattern.mockResolvedValue([])

      const result = await searchPattern({})

      expect(result).toEqual([])
    })
  })

  describe('getDeviceVersions', () => {
    it('возвращает версии устройства', async () => {
      const mockData = { versions: [] }
      getDeviceVersions.mockResolvedValue(mockData)

      const result = await getDeviceVersions('device-123')

      expect(result).toEqual(mockData)
    })
  })

  describe('getLatestVersionsForDevices', () => {
    it('возвращает последние версии двух устройств', async () => {
      const mockData = { left: {}, right: {} }
      getLatestVersionsForDevices.mockResolvedValue(mockData)

      const result = await getLatestVersionsForDevices('left-id', 'right-id')

      expect(result).toEqual(mockData)
    })
  })

  describe('getDevicesDiff', () => {
    it('возвращает diff двух устройств', async () => {
      const mockData = { diff: [] }
      getDevicesDiff.mockResolvedValue(mockData)

      const result = await getDevicesDiff('dev1', 'dev2', '2024-01-01')

      expect(result).toEqual(mockData)
    })
  })

  describe('getDiffByDate', () => {
    it('возвращает diff по датам', async () => {
      const mockData = { diff: [] }
      getDiffByDate.mockResolvedValue(mockData)

      const result = await getDiffByDate('dev-123', '2024-01-01', '2024-01-02')

      expect(result).toEqual(mockData)
    })

    it('возвращает ошибку если версии не найдены', async () => {
      const mockError = { error: 'no versions found for date 2024-01-01' }
      getDiffByDate.mockRejectedValue(new Error(JSON.stringify(mockError)))

      await expect(getDiffByDate('dev-123', '2024-01-01', '2024-01-02'))
        .rejects.toThrow()
    })

    it('возвращает diff между двумя версиями', async () => {
      const mockDiff = {
        left_version_id: 10,
        right_version_id: 15,
        added_lines: 5,
        removed_lines: 2,
        diff: '---old\n+++new\n@@ -1,2 +1,3 @@',
      }
      getDiffByDate.mockResolvedValue(mockDiff)

      const result = await getDiffByDate('dev-123', '2024-01-15', '2024-01-20')

      expect(result.left_version_id).toBe(10)
      expect(result.right_version_id).toBe(15)
      expect(result.added_lines).toBe(5)
      expect(result.removed_lines).toBe(2)
    })

    it('принимает пустые строки дат', async () => {
      const mockData = { diff: [] }
      getDiffByDate.mockResolvedValue(mockData)

      const result = await getDiffByDate('dev-123', '', '')

      expect(result).toEqual(mockData)
    })

    it('валидирует deviceId', async () => {
      const mockError = { error: 'invalid deviceId' }
      getDiffByDate.mockRejectedValue(new Error(JSON.stringify(mockError)))

      await expect(getDiffByDate('', '2024-01-01', '2024-01-02'))
        .rejects.toThrow()
    })
  })

  describe('exportConfigByDate', () => {
    it('возвращает blob с конфигом', async () => {
      const mockResponse = { data: new Blob(), status: 200 }
      exportConfigByDate.mockResolvedValue(mockResponse)

      const result = await exportConfigByDate('dev-123', '2024-01-01')

      expect(result).toEqual(mockResponse)
    })
  })

  describe('searchChanges', () => {
    it('возвращает изменения', async () => {
      const mockData = { changes: [] }
      searchChanges.mockResolvedValue(mockData)

      const result = await searchChanges({ deviceId: 'dev-123' })

      expect(result).toEqual(mockData)
    })
  })

  describe('triggerScan', () => {
    it('запускает сканирование', async () => {
      const mockData = { status: 'started' }
      triggerScan.mockResolvedValue(mockData)

      const result = await triggerScan()

      expect(result).toEqual(mockData)
    })
  })

  describe('getScanStatus', () => {
    it('возвращает статус сканирования', async () => {
      const mockData = { status: 'running', progress: 50 }
      getScanStatus.mockResolvedValue(mockData)

      const result = await getScanStatus()

      expect(result).toEqual(mockData)
    })
  })
})

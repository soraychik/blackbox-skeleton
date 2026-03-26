import { describe, it, expect } from 'vitest'
import { toDDMMYYYY, formatDateTime } from './dateFormatter'

// ============================================
// Тесты для функции toDDMMYYYY
// Конвертирует дату из формата YYYY-MM-DD в DD/MM/YYYY
// ============================================
describe('toDDMMYYYY', () => {
  // Тест: проверяет базовую конвертацию даты
  it('конвертирует YYYY-MM-DD в DD/MM/YYYY', () => {
    expect(toDDMMYYYY('2024-03-15')).toBe('15/03/2024')
  })

  // Тест: проверяет обработку null и undefined
  // Ожидаем пустую строку, если передан null или undefined
  it('возвращает пустую строку для null и undefined', () => {
    expect(toDDMMYYYY(null)).toBe('')
    expect(toDDMMYYYY(undefined)).toBe('')
  })

  // Тест: проверяет обработку невалидного формата
  // Если формат не соответствует ожидаемому, возвращаем исходную строку
  it('возвращает исходную строку для невалидного формата', () => {
    expect(toDDMMYYYY('invalid')).toBe('invalid')
    expect(toDDMMYYYY('2024')).toBe('2024')
  })

  // Тест: проверяет добавление ведущих нулей для дней и месяцев
  // Например: 05 января должно стать "05/01/2024"
  it('добавляет ведущий ноль для однозначных значений', () => {
    expect(toDDMMYYYY('2024-01-05')).toBe('05/01/2024')
  })
})

// ============================================
// Тесты для функции formatDateTime
// Форматирует дату в формат DD/MM/YYYY HH:MM:SS
// ============================================
describe('formatDateTime', () => {
  // Тест: проверяет форматирование ISO даты с временем
  // Ожидаемый формат: "15/03/2024 10:30:45"
  it('форматирует ISO дату со временем корректно', () => {
    const result = formatDateTime('2024-03-15T10:30:45')
    expect(result).toBe('15/03/2024 10:30:45')
  })

  // Тест: проверяет обработку пустых значений
  // Ожидаем пустую строку для null, undefined или пустой строки
  it('возвращает пустую строку для null, undefined и пустой строки', () => {
    expect(formatDateTime(null)).toBe('')
    expect(formatDateTime(undefined)).toBe('')
    expect(formatDateTime('')).toBe('')
  })

  // Тест: проверяет обработку невалидной даты
  // Если Date не может распарсить строку, возвращаем исходную
  it('возвращает исходную строку для невалидной даты', () => {
    expect(formatDateTime('invalid-date')).toBe('invalid-date')
  })
})

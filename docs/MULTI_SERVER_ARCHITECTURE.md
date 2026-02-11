# Multi-Server File Architecture

## Overview

BlackBox Scheduler теперь поддерживает множественные файловые серверы с возможностью масштабирования и автоматического управления.

## Architecture Components

### 1. FileServerManager
Основной менеджер для управления множественными файловыми серверами:
- Параллельная обработка серверов
- Health checking
- Автоматическое восстановление соединений
- Graceful shutdown

### 2. FileServerInstance
Экземпляр файлового сервера:
- Конфигурация сервера
- Состояние подключения
- Last check timestamp

### 3. RemoteFileSystem
Абстракция над удаленными файловыми системами:
- NFS support
- SMB/CIFS support
- Local filesystem support
- Mount/unmount operations

## Configuration System

### Environment Variables Pattern
```
# Server 1 (primary)
FILE_SERVER_TYPE=nfs
NFS_SERVER=192.168.50.149
NFS_SHARE_PATH=/srv/share
NFS_MOUNT_POINT=/mnt/nfs

# Server 2 (secondary)
FILE_SERVER_2_ENABLED=true
FILE_SERVER_2_TYPE=smb
FILE_SERVER_2_SERVER=192.168.50.150
FILE_SERVER_2_SHARE_NAME=share
FILE_SERVER_2_MOUNT_POINT=/mnt/smb2
FILE_SERVER_2_USERNAME=user2
FILE_SERVER_2_PASSWORD=pass2
```

### Scaling Pattern
Для добавления нового сервера:
1. Добавить переменные `FILE_SERVER_{N}_ENABLED=true`
2. Добавить специфичные переменные с префиксом `FILE_SERVER_{N}_`
3. Обновить `getConfigsFromEnv()` в `manager.go`

## Processing Flow

### Startup Sequence
1. Load server configurations from environment
2. Initialize FileServerManager
3. Mount all enabled servers
4. Start processing loop

### Runtime Processing
```
┌─────────────────┐
│   Main Loop     │
│   (30 sec)      │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│ ProcessAllServers│
│   (parallel)    │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│ Server 1        │
│ GetFiles()      │
│ ProcessFile()   │
└─────────────────┘
┌─────────────────┐
│ Server 2        │
│ GetFiles()      │
│ ProcessFile()   │
└─────────────────┘
┌─────────────────┐
│ Server N        │
│ GetFiles()      │
│ ProcessFile()   │
└─────────────────┘
```

### Health Checking
```
┌─────────────────┐
│ Health Loop     │
│   (60 sec)      │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│ CheckHealth()   │
│   (sequential)  │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│ Server Healthy? │
│      Yes        │
└─────────────────┘
          │
          ▼
┌─────────────────┐
│ Continue        │
└─────────────────┘

┌─────────────────┐
│ Server Healthy? │
│      No         │
└─────────┬───────┘
          │
          ▼
┌─────────────────┐
│ Attempt Remount │
│   + Healthcheck │
└─────────────────┘
```

## Device Naming Convention

Для обеспечения уникальности:
- `{serverID}-{filename}`
- Примеры:
  - `server1-config.json`
  - `server2-config.json`
  - `local-config.json`

## Error Handling

### Mount Failures
- Логирование ошибок
- Продолжение работы с другими серверами
- Попытки перемонтирования при health check

### File Processing Errors
- Изолированные ошибки не влияют на другие серверы
- Детальное логирование
- Continue-on-error стратегия

### Connection Failures
- Автоматическое перемонтирование
- Graceful degradation
- Health monitoring

## Performance Considerations

### Parallel Processing
- Каждый сервер обрабатывается в отдельной goroutine
- I/O операции не блокируют другие серверы
- Configurable concurrency

### Resource Management
- Graceful shutdown
- Proper unmounting
- Memory-efficient file processing

### Caching
- File system caching
- Connection pooling
- Configurable timeouts

## Future Enhancements

### Dynamic Configuration
- Runtime server addition/removal
- Configuration via API
- Hot reloading

### Load Balancing
- Server weighting
- Intelligent failover
- Geographic distribution

### Advanced Monitoring
- Metrics collection
- Performance analytics
- Alerting

## Example Configurations

### Multiple NFS Servers
```env
FILE_SERVER_TYPE=nfs
NFS_SERVER=192.168.50.149
NFS_SHARE_PATH=/srv/configs
NFS_MOUNT_POINT=/mnt/nfs1

FILE_SERVER_2_ENABLED=true
FILE_SERVER_2_TYPE=nfs
FILE_SERVER_2_SERVER=192.168.50.150
FILE_SERVER_2_SHARE_PATH=/srv/configs
FILE_SERVER_2_MOUNT_POINT=/mnt/nfs2

FILE_SERVER_3_ENABLED=true
FILE_SERVER_3_TYPE=nfs
FILE_SERVER_3_SERVER=192.168.50.151
FILE_SERVER_3_SHARE_PATH=/srv/configs
FILE_SERVER_3_MOUNT_POINT=/mnt/nfs3
```

### Mixed Environment
```env
# Primary NFS
FILE_SERVER_TYPE=nfs
NFS_SERVER=192.168.50.149
NFS_SHARE_PATH=/srv/configs
NFS_MOUNT_POINT=/mnt/nfs

# Secondary SMB
FILE_SERVER_2_ENABLED=true
FILE_SERVER_2_TYPE=smb
FILE_SERVER_2_SERVER=192.168.50.150
FILE_SERVER_2_SHARE_NAME=configs
FILE_SERVER_2_MOUNT_POINT=/mnt/smb2
FILE_SERVER_2_USERNAME=admin
FILE_SERVER_2_PASSWORD=secret123

# Local fallback
LOCAL_FS_ENABLED=true
LOCAL_FS_PATH=/app/configs
```

## Migration Guide

### From Single Server to Multi-Server
1. Backup current configuration
2. Update .env with new variables
3. Update docker-compose.yml if needed
4. Test with one additional server
5. Gradually add more servers

### Best Practices
- Start with 2-3 servers
- Monitor performance closely
- Use consistent naming conventions
- Document server purposes
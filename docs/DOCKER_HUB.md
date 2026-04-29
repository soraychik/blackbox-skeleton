# Запуск BlackBox через Docker Hub

Инструкция в двух частях: **как выложить образы** (разработчик) и **как запустить у себя** (руководитель или другой пользователь).

---

## Часть 1. Разработчик: публикация образов в Docker Hub

### 1.1 Регистрация и вход

1. Зарегистрируйтесь на [Docker Hub](https://hub.docker.com) (бесплатный аккаунт).
2. В терминале выполните вход:
   ```bash
   docker login
   ```
   Введите логин и пароль (или токен доступа). Запомните **логин** — он понадобится для тегов и для руководителя.

### 1.2 Сборка образов

В корне проекта:

```bash
cd blackbox
docker compose build
```

После сборки появятся три образа (имена зависят от имени папки проекта, обычно с префиксом `blackbox-`):

- `blackbox-api`
- `blackbox-scheduler`
- `blackbox-nginx`

Проверить: `docker images | grep blackbox`

### 1.3 Тегирование под Docker Hub

Замените `ВАШ_ЛОГИН` на ваш логин Docker Hub.

```bash
export DOCKERHUB_USER=soraychik

docker tag blackbox-api:latest    $DOCKERHUB_USER/blackbox-api:latest
docker tag blackbox-scheduler:latest  $DOCKERHUB_USER/blackbox-scheduler:latest
docker tag blackbox-nginx:latest       $DOCKERHUB_USER/blackbox-nginx:latest
```

Если имена образов у вас другие (например без `blackbox-`), подставьте их в команды выше. Узнать точные имена: `docker images`.

### 1.4 Публикация в Docker Hub

```bash
docker push $DOCKERHUB_USER/blackbox-api:latest
docker push $DOCKERHUB_USER/blackbox-scheduler:latest
docker push $DOCKERHUB_USER/blackbox-nginx:latest
```

После этого образы доступны по адресам вида:
- `ВАШ_ЛОГИН/blackbox-api:latest`
- `ВАШ_ЛОГИН/blackbox-scheduler:latest`
- `ВАШ_ЛОГИН/blackbox-nginx:latest`

### 1.5 Что передать руководителю

1. **Ссылку на репозиторий проекта** (git или архив с папкой `blackbox`), чтобы у него были:
   - `docker-compose.hub.yml`
   - `.env.example`
   - папки `sql-init` и `scheduler/configs`
2. **Ваш логин Docker Hub** — руководитель пропишет его в `.env` как `DOCKERHUB_USER=ВАШ_ЛОГИН`.
3. **Краткую инструкцию** — например: «Скопируй `.env.example` в `.env`, укажи в нём `DOCKERHUB_USER=мой_логин`, затем выполни команды из раздела 2 ниже».

---

## Часть 2. Руководитель (или другой пользователь): запуск проекта

Требования: установленные **Docker** и **Docker Compose** (v2).

### 2.1 Получить проект

- **Через Git:**  
  `git clone <URL репозитория>`  
  затем: `cd blackbox`

- **Через архив:**  
  распаковать архив в папку и перейти в неё:  
  `cd blackbox`

Убедитесь, что в каталоге есть файлы:
- `docker-compose.hub.yml`
- `.env.example`
- папка `sql-init` с файлом `init.sql`
- папка `scheduler/configs`

### 2.2 Создать конфигурацию .env

```bash
cp .env.example .env
```

Откройте `.env` и **обязательно** укажите логин Docker Hub разработчика (тот, под которым выложены образы):

```env
DOCKERHUB_USER=логин_разработчика_на_docker_hub
```

Остальные переменные в `.env.example` уже подходят для запуска «из коробки» (MySQL и MinIO поднимаются в контейнерах). При необходимости можно изменить другие параметры (MinIO, файловые серверы и т.д.) — см. README проекта.

### 2.3 Скачать образы и запустить контейнеры

```bash
docker compose -f docker-compose.hub.yml pull
docker compose -f docker-compose.hub.yml up -d
```

Команда `pull` скачает образы с Docker Hub, `up -d` запустит все сервисы в фоне.

### 2.4 Проверить запуск

```bash
docker compose -f docker-compose.hub.yml ps
```

Все сервисы (mysql-db, minio, scheduler, api, nginx) должны быть в состоянии `Up`.

Откройте в браузере: **http://localhost**  
Должен открыться веб-интерфейс BlackBox (через nginx на порту 80).

### 2.5 Остановка и удаление

```bash
docker compose -f docker-compose.hub.yml down
```

Данные MySQL и MinIO сохраняются в томах Docker и не удаляются при `down`. Чтобы удалить и их: `docker compose -f docker-compose.hub.yml down -v` (осторожно — все данные БД и хранилища будут потеряны).

---

## Краткая шпаргалка для руководителя

```bash
cd blackbox
cp .env.example .env
# В .env указать: DOCKERHUB_USER=логин_разработчика
docker compose -f docker-compose.hub.yml pull
docker compose -f docker-compose.hub.yml up -d
# Открыть в браузере: http://localhost
```

После этого всё должно заработать без сборки исходного кода.

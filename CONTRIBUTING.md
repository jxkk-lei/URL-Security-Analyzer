# Contributing

Правила разработки для **URL Security Analyzer**.

## 1. Branches

Не работайте напрямую в `main`.

Для каждой задачи создавайте отдельную ветку:

```text
feature/<name>    — новая функциональность
fix/<name>        — исправление ошибки
refactor/<name>   — переработка кода
test/<name>       — тесты
docs/<name>       — документация
chore/<name>      — технические изменения
```

Примеры:

```text
feature/url-analysis
feature/authentication
fix/url-validation
refactor/analyzer
docs/api
```

После завершения работы создавайте Pull Request в `main`.

## 2. Naming — Go

Используем стандартные соглашения Go.

* Переменные и функции: `camelCase`
* Экспортируемые функции и типы: `PascalCase`
* Пакеты: `lowercase`
* Файлы: `snake_case.go`
* Аббревиатуры пишутся заглавными: `ID`, `URL`, `HTTP`, `API`, `TLS`, `JSON`


## 3. Naming — TypeScript

Используем стандартные соглашения TypeScript/React.

* Переменные и функции: `camelCase`
* Компоненты React: `PascalCase`
* Интерфейсы и типы: `PascalCase`
* Классы: `PascalCase`
* Константы: `UPPER_SNAKE_CASE` для глобальных/конфигурационных значений
* Файлы компонентов: `PascalCase.tsx`
* Обычные файлы: `camelCase.ts`


## 4. Commits

Используем Conventional Commits:

```text
feat: add URL analysis
fix: validate URL format
refactor: improve analyzer service
test: add URL validation tests
docs: update README
chore: update dependencies
```

Один commit должен содержать одно логически связанное изменение.

## 5. Pull Requests

Pull Request должен:

* иметь понятное название;
* содержать описание изменений;
* содержать только связанные с задачей изменения;
* проходить тесты;
* не содержать секретов.

Перед PR обязательно проверьте что

* код работает 
* тесты проходят
* нет секретов
* naming соответствует правилам 

## 6. Code

* Соблюдайте существующую архитектуру проекта.
* Не дублируйте код без необходимости.
* Не смешивайте несколько независимых задач.
* Перед добавлением нового компонента проверьте, нет ли уже существующего решения.
* Не оставляйте закомментированный или отладочный код без необходимости.

## 7. Security

Не добавляйте в Git:

```text
.env
пароли
API keys
JWT secrets
database credentials
private keys
```

Для примера конфигурации используйте `.env.example`.

## 8. General Rules

* `main` используется только для стабильного кода.
* Все изменения `main` выполняются через Pull Request.
* Новая функциональность должна иметь соответствующие тесты.
* Изменения API должны быть отражены в документации.
* Не изменяйте чужие части проекта без необходимости.

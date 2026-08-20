# Task Tracker — full-stack темплейт (Go + React)

Базовий темплейт застосунку: модульний Go-моноліт + повна інфраструктура.
Продуктових фіч немає навмисно — це стартова точка, з якої фічі наростають
(напр., через SDD-пайплайн). Головна сторінка — гола кнопка Google-логіну,
дашборд — голе «Hello»: дизайн будується вже у твоєму проєкті.

- **api/** — Go: chi + pgx + golang-migrate, Google OAuth + JWT, `/livez` `/readyz`
  `/metrics`, testcontainers-інтеграційні тести. Чартер: `api/CLAUDE.md`.
<!-- battery:web -->
- **web/** — React Router 7 SPA (ssr:false): Tailwind 4 + shadcn-примітиви, FSD,
  Google-логін, vitest + Playwright.
<!-- /battery:web -->
- **docker-compose.yml** — повний локальний стек (`make up`).
<!-- battery:ci -->
- **.github/workflows/ci.yml** — CI: ті самі перевірки, що й `make check` локально.
<!-- /battery:ci -->
<!-- battery:deploy -->
- **.github/workflows/deploy.yml** + **deploy/** — деплой: GHCR-образи → VPS →
  міграції → health-гейт; Caddy з авто-TLS.
<!-- /battery:deploy -->
<!-- battery:observability -->
- **Prometheus + Grafana** — скрейп `/metrics` локально і в проді; дашборд у
  `deploy/grafana/dashboards/`.
<!-- /battery:observability -->
- **.claude/** — harness: path-rules для Go, gofmt-хук, go-скіли, агенти-ревʼюери.
  Кореневого CLAUDE.md немає навмисно — згенеруй його `/init` у своєму проєкті.

## Швидкий старт

```bash
cp api/.env.example api/.env   # dummy-значення; Google-логін потребує реальних CLIENT_ID/SECRET
make up                                      # повний локальний стек
make check                                   # наскрізна перевірка
```

- API: http://localhost:8080 (`/livez`, `/readyz`, `/metrics`)
<!-- battery:web -->
- Web: http://localhost:5173
<!-- /battery:web -->
<!-- battery:observability -->
- Grafana: http://localhost:3000 · Prometheus: http://localhost:9090
<!-- /battery:observability -->

Дев-цикл без docker: `make -C api run` (Postgres лишається з compose).
<!-- battery:web -->
Фронт у дев-циклі: `cd web && npm run dev`.
<!-- /battery:web -->

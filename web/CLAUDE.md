# task-tracker/web — Frontend charter

Thin, always-on charter for the React SPA. Sets posture and indexes conventions;
the design system lives in code (`src/shared/ui` + Tailwind tokens), not here.

## What this is
- React 19 + **React Router 7 SPA** (`ssr: false`) · TypeScript · Vite.
- **FSD (Feature-Sliced Design)**: `app/ → pages/ → widgets/ → features/ → entities/ → shared/`.
  Імпорти лише «вниз» по шарах; крос-імпорти між slice-ами одного шару заборонені.
- UI: Tailwind 4 + shadcn-примітиви у `src/shared/ui` (не редагуй згенеровані примітиви —
  компонуй поверх них).
- API-клієнт: `src/shared/api/client.ts` (typed fetch, обробка помилок через `showApiError`).
- Auth: Google OAuth через `features/auth-by-google` + `app/providers/auth` (JWT у пам'яті).

## Working posture
- **TDD**: vitest + Testing Library поруч із кодом (`*.test.tsx`); e2e/smoke — Playwright у `e2e/`.
- Команди: `npm run typecheck` · `npm test` · `npm run test:e2e:smoke` (потрібен живий стек `make up`).
- Нова фіча = новий slice у `features/` (api/model/ui), сторінка збирає slices; логіку в pages не класти.
- Роути: `src/routes.ts`; захищені — під `app/layouts/ProtectedLayout`.

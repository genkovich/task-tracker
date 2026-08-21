# Tracker — tasks

> Статус кожної задачі епіка. States: `todo` · `in_progress` · `blocked` · `review` · `done`.

| # | Task | Layer | Owner | Estimate | Blocked by | Status |
|---|---|---|---|---|---|---|
| T1 | Promote the staged migrations into the live tree | migration | genkovich | S | — | done |
| T2 | Extend the task domain + Comment entity | domain | genkovich | M | — | done |
| T3 | Persist details and comments; card fields in board state | infra | genkovich | L | T1, T2 | done |
| T4 | Task use-cases + task-detail reads (editor and token-scoped) | app | genkovich | M | T2, T3 | done |
| T5 | Comment use-cases with board-scoped broadcast | app | genkovich | S | T2, T3 | done |
| T6 | Task detail GET + нові поля на create/edit | ports | genkovich | M | T4 | done |
| T7 | Comment routes (list, add, delete) | ports | genkovich | S | T5 | done |
| T8 | Public task detail у тирі 300/хв + wiring | ports | genkovich | M | T4, T6, T7 | done |
| T9 | Web API layer (типи + клієнт) | ui | genkovich | S | T6, T7, T8 | done |
| T10 | Маркери на картці | ui | genkovich | M | T9 | done |
| T11 | Детальна модалка редактора + read-only глядача | ui | genkovich | L | T9, T10 | done |
| T12 | E2E наскрізь + CHANGELOG | ui | genkovich | M | T10, T11 | done |

**Total:** 12 задач, ~5–6 людино-днів. Усі на гілці `feat/task-details`.

## Known deviations / follow-ups

- **`EditTaskModal` замінено, а не розширено.** Модалка виросла з двох полів до пʼяти
  плюс стрічка коментарів і форма, і мусила почати з мережевого читання деталей — це вже
  інший компонент. `TaskDetailsModal.tsx` (+ `TaskComments.tsx`) прийшли на місце
  `EditTaskModal.tsx`, який видалено разом із його тестом; поведінку Delete збережено
  один-в-один.
- **`ColumnState.Tasks` змінив тип** із `[]domain.Task` на `[]ports.TaskListItem`: стан
  дошки більше не є списком доменних задач, бо свідомо не несе опису, зате несе два
  похідні поля. Це зачепило всі місця, що читали стан дошки.
- **Публічні деталі перевіряють дошку токена в тому ж читанні**, що й резолвінг задачі
  (T4), а не окремим запитом після — інакше між перевіркою й читанням лишалась би щілина.

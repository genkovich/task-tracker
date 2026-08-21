import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { TaskDetailsModal } from "./TaskDetailsModal";
import type { Task, TaskDetail } from "@/features/board/api/types";

const mockGetTask = vi.fn();
const mockGetPublicTask = vi.fn();
const mockEditTask = vi.fn();
const mockDeleteTask = vi.fn();
const mockAddComment = vi.fn();
const mockDeleteComment = vi.fn();

vi.mock("@/features/board/api/boardApi", () => ({
  boardApi: {
    getTask: (...args: unknown[]) => mockGetTask(...args),
    getPublicTask: (...args: unknown[]) => mockGetPublicTask(...args),
    editTask: (...args: unknown[]) => mockEditTask(...args),
    deleteTask: (...args: unknown[]) => mockDeleteTask(...args),
    addComment: (...args: unknown[]) => mockAddComment(...args),
    deleteComment: (...args: unknown[]) => mockDeleteComment(...args),
  },
}));

// The comment form pre-fills the author from the signed-in person — free
// text, not an identity (ADR-0001). Stubbed here so the field has a value to
// assert.
vi.mock("@/app/providers/auth", () => ({
  useAuth: () => ({
    user: { first_name: "Ada", last_name: "Lovelace", email: "ada@example.com" },
  }),
}));

const mockShowApiError = vi.fn();
vi.mock("@/shared/lib/showApiError", () => ({
  showApiError: (...args: unknown[]) => mockShowApiError(...args),
}));

const card: Task = {
  id: "task-1",
  column_id: "col-1",
  title: "Написати звіт",
  assignee: "Alex",
  priority: "high",
  due_date: "2026-09-01",
  has_description: true,
  comment_count: 1,
  created_at: "2026-08-20T00:00:00Z",
  updated_at: "2026-08-20T00:00:00Z",
};

const detail: TaskDetail = {
  task: {
    id: "task-1",
    column_id: "col-1",
    title: "Написати звіт",
    assignee: "Alex",
    description: "Зібрати цифри за тиждень",
    priority: "high",
    due_date: "2026-09-01",
    created_at: "2026-08-20T00:00:00Z",
    updated_at: "2026-08-20T00:00:00Z",
  },
  comments: [
    {
      id: "comment-1",
      task_id: "task-1",
      author: "Grace",
      body: "Цифри вже є в дашборді",
      created_at: "2026-08-21T09:00:00Z",
    },
  ],
};

function renderEditor(overrides: Partial<React.ComponentProps<typeof TaskDetailsModal>> = {}) {
  return render(
    <TaskDetailsModal
      task={card}
      open
      onOpenChange={() => {}}
      onSaved={() => {}}
      onDeleted={() => {}}
      {...overrides}
    />,
  );
}

beforeEach(() => {
  vi.clearAllMocks();
  mockGetTask.mockResolvedValue(detail);
  mockGetPublicTask.mockResolvedValue(detail);
});

describe("TaskDetailsModal — editor (SCR-03)", () => {
  it("loads the detail and pre-fills every field", async () => {
    renderEditor();

    expect(await screen.findByDisplayValue("Написати звіт")).toBeInTheDocument();
    expect(mockGetTask).toHaveBeenCalledWith("task-1");
    expect(screen.getByLabelText("Опис")).toHaveValue("Зібрати цифри за тиждень");
    expect(screen.getByLabelText("Пріоритет")).toHaveValue("high");
    expect(screen.getByLabelText("Дедлайн")).toHaveValue("2026-09-01");
    expect(screen.getByLabelText("Виконавець")).toHaveValue("Alex");
  });

  // TSK-01/TSK-03/TSK-05: every detail field reaches the API on save.
  it("saves the edited description, priority and deadline", async () => {
    const user = userEvent.setup();
    mockEditTask.mockResolvedValue(detail.task);
    const onSaved = vi.fn();

    renderEditor({ onSaved });
    await screen.findByDisplayValue("Написати звіт");

    await user.clear(screen.getByLabelText("Опис"));
    await user.type(screen.getByLabelText("Опис"), "Новий опис");
    await user.selectOptions(screen.getByLabelText("Пріоритет"), "low");
    await user.clear(screen.getByLabelText("Дедлайн"));
    await user.type(screen.getByLabelText("Дедлайн"), "2026-10-05");

    await user.click(screen.getByRole("button", { name: "Зберегти" }));

    await waitFor(() => {
      expect(mockEditTask).toHaveBeenCalledWith("task-1", {
        title: "Написати звіт",
        assignee: "Alex",
        description: "Новий опис",
        priority: "low",
        due_date: "2026-10-05",
      });
    });
    expect(onSaved).toHaveBeenCalled();
  });

  // TSK-07: clearing the deadline field sends null, not an empty string the
  // API would reject as a malformed date.
  it("clearing the deadline sends null", async () => {
    const user = userEvent.setup();
    mockEditTask.mockResolvedValue(detail.task);

    renderEditor();
    await screen.findByDisplayValue("Написати звіт");

    await user.clear(screen.getByLabelText("Дедлайн"));
    await user.click(screen.getByRole("button", { name: "Зберегти" }));

    await waitFor(() => {
      expect(mockEditTask).toHaveBeenCalledWith(
        "task-1",
        expect.objectContaining({ due_date: null }),
      );
    });
  });

  it("saving with an empty title shows the inline error and does not call the API", async () => {
    const user = userEvent.setup();

    renderEditor();
    await screen.findByDisplayValue("Написати звіт");

    await user.clear(screen.getByLabelText("Назва"));
    await user.click(screen.getByRole("button", { name: "Зберегти" }));

    expect(await screen.findByText(/назва обов'язкова/i)).toBeInTheDocument();
    expect(mockEditTask).not.toHaveBeenCalled();
  });

  // The fields start from the card, which carries no description. If Save
  // were reachable before the detail arrived, it would send an empty
  // description and quietly wipe the real one.
  it("offers no save until the detail has actually loaded", async () => {
    mockGetTask.mockRejectedValue(new Error("boom"));

    renderEditor();

    expect(await screen.findByText(/не вдалося завантажити деталі/i)).toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Зберегти" })).not.toBeInTheDocument();
    expect(mockEditTask).not.toHaveBeenCalled();
  });

  // Delete behaves exactly as it did in the modal this one replaced.
  it("deleting the task reports it to the caller", async () => {
    const user = userEvent.setup();
    mockDeleteTask.mockResolvedValue(undefined);
    const onDeleted = vi.fn();

    renderEditor({ onDeleted });
    await screen.findByDisplayValue("Написати звіт");

    await user.click(screen.getByRole("button", { name: "Видалити" }));

    await waitFor(() => {
      expect(mockDeleteTask).toHaveBeenCalledWith("task-1");
      expect(onDeleted).toHaveBeenCalledWith("task-1");
    });
  });
});

describe("TaskDetailsModal — comments (SCR-03)", () => {
  it("shows the existing thread", async () => {
    renderEditor();

    expect(await screen.findByText("Цифри вже є в дашборді")).toBeInTheDocument();
    expect(screen.getByText("Grace")).toBeInTheDocument();
  });

  // TSK-08: the author field starts filled with the signed-in person's name,
  // and a valid comment reaches the API.
  it("pre-fills the author and posts a new comment", async () => {
    const user = userEvent.setup();
    mockAddComment.mockResolvedValue({});

    renderEditor();
    await screen.findByDisplayValue("Написати звіт");

    expect(screen.getByLabelText("Автор")).toHaveValue("Ada Lovelace");

    await user.type(screen.getByLabelText("Новий коментар"), "Готово");
    await user.click(screen.getByRole("button", { name: "Додати коментар" }));

    await waitFor(() => {
      expect(mockAddComment).toHaveBeenCalledWith("task-1", {
        author: "Ada Lovelace",
        body: "Готово",
      });
    });
    // The thread is re-read so the new comment appears without a page reload.
    expect(mockGetTask).toHaveBeenCalledTimes(2);
  });

  // TSK-09: an empty comment is blocked in the form, before any request.
  it("blocks an empty comment without calling the API", async () => {
    const user = userEvent.setup();

    renderEditor();
    await screen.findByDisplayValue("Написати звіт");

    await user.click(screen.getByRole("button", { name: "Додати коментар" }));

    expect(await screen.findByText(/коментар не може бути порожнім/i)).toBeInTheDocument();
    expect(mockAddComment).not.toHaveBeenCalled();
  });

  it("blocks a comment with no author", async () => {
    const user = userEvent.setup();

    renderEditor();
    await screen.findByDisplayValue("Написати звіт");

    await user.clear(screen.getByLabelText("Автор"));
    await user.type(screen.getByLabelText("Новий коментар"), "Готово");
    await user.click(screen.getByRole("button", { name: "Додати коментар" }));

    expect(await screen.findByText(/вкажіть автора/i)).toBeInTheDocument();
    expect(mockAddComment).not.toHaveBeenCalled();
  });

  // TSK-10.
  it("deletes a comment", async () => {
    const user = userEvent.setup();
    mockDeleteComment.mockResolvedValue(undefined);

    renderEditor();
    await screen.findByText("Цифри вже є в дашборді");

    await user.click(screen.getByRole("button", { name: /видалити коментар від grace/i }));

    await waitFor(() => {
      expect(mockDeleteComment).toHaveBeenCalledWith("task-1", "comment-1");
    });
  });
});

// TSK-12: the viewer's dialog is read-only as a property of its markup, not
// as a flag some handler checks — there is nothing to type into and nothing
// to press that would change anything.
describe("TaskDetailsModal — viewer (SCR-07)", () => {
  it("fetches through the public token and renders the same content as text", async () => {
    renderEditor({ publicToken: "tok-123" });

    expect(await screen.findByText("Зібрати цифри за тиждень")).toBeInTheDocument();
    expect(mockGetPublicTask).toHaveBeenCalledWith("tok-123", "task-1");
    expect(mockGetTask).not.toHaveBeenCalled();

    // The same thread the editor sees.
    expect(screen.getByText("Цифри вже є в дашборді")).toBeInTheDocument();
    expect(screen.getByText("Високий")).toBeInTheDocument();
    expect(screen.getByText("1 вер")).toBeInTheDocument();
  });

  it("contains no field, no comment form and no delete", async () => {
    const { container } = renderEditor({ publicToken: "tok-123" });
    await screen.findByText("Зібрати цифри за тиждень");

    expect(container.querySelector("input")).toBeNull();
    expect(container.querySelector("textarea")).toBeNull();
    expect(container.querySelector("select")).toBeNull();
    expect(screen.queryByRole("button", { name: "Зберегти" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Видалити" })).not.toBeInTheDocument();
    expect(screen.queryByRole("button", { name: "Додати коментар" })).not.toBeInTheDocument();
    expect(
      within(screen.getByRole("region", { name: "Коментарі" })).queryByRole("button", {
        name: /видалити коментар/i,
      }),
    ).not.toBeInTheDocument();
  });
});

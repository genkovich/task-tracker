import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { ApiClientError, BASE_URL } from "@/shared/api/client";
import { boardApi } from "./boardApi";

function jsonResponse(body: unknown, status = 200) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  };
}

beforeEach(() => {
  localStorage.clear();
});

afterEach(() => {
  vi.restoreAllMocks();
  vi.unstubAllGlobals();
});

describe("boardApi.getBoard", () => {
  it("GETs the board state from /api/v1/boards/{boardId}", async () => {
    const board = { id: "board-1", name: "Дошка команди", columns: [], public_link: null };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(board));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.getBoard("board-1");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/boards/board-1`);
    expect(options?.method ?? "GET").toBe("GET");
    expect(result).toEqual(board);
  });
});

describe("boardApi.listBoards", () => {
  it("GETs the dashboard rows from /api/v1/boards", async () => {
    const boards = [
      { id: "board-1", name: "Дошка команди", created_at: "2026-08-20T00:00:00Z", task_count: 3 },
    ];
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(boards));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.listBoards();

    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/boards`);
    expect(options?.method ?? "GET").toBe("GET");
    expect(result).toEqual(boards);
  });
});

describe("boardApi.createBoard", () => {
  it("POSTs the name to /api/v1/boards and resolves with the new board", async () => {
    const created = {
      id: "board-2",
      name: "Воркшоп",
      created_at: "2026-08-21T00:00:00Z",
      columns: [],
      public_link: null,
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(created, 201));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.createBoard("Воркшоп");

    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/boards`);
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({ name: "Воркшоп" });
    expect(result).toEqual(created);
  });

  it("maps a 422 empty-name error to an ApiClientError", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse(
          { error: { code: "board.name_required", message: "board name is required" } },
          422,
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.createBoard("")).rejects.toMatchObject({
      code: "board.name_required",
      statusCode: 422,
    });
  });
});

// Review 2026-08-21 root K: token/taskId are interpolated into URL paths —
// a value with reserved characters must not break out of its path segment.
describe("boardApi — path parameters are URL-encoded", () => {
  it("encodes the public token in the board path", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({ columns: [] }));
    vi.stubGlobal("fetch", fetchMock);

    await boardApi.getPublicBoard("tok/../evil?x=1");

    const [url] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/public/${encodeURIComponent("tok/../evil?x=1")}/board`);
  });

  it("encodes the taskId in task paths", async () => {
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse({}));
    vi.stubGlobal("fetch", fetchMock);

    await boardApi.editTask("id with/slash", { title: "x" });

    const [url] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks/${encodeURIComponent("id with/slash")}`);
  });
});

describe("boardApi.createTask", () => {
  it("POSTs board_id + title/assignee to /api/v1/tasks", async () => {
    const created = {
      id: "task-1",
      column_id: "col-1",
      title: "Test task",
      assignee: "Test User",
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:00:00Z",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(created, 201));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.createTask({
      board_id: "board-1",
      title: "Test task",
      assignee: "Test User",
    });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks`);
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({
      board_id: "board-1",
      title: "Test task",
      assignee: "Test User",
    });
    expect(result).toEqual(created);
  });

  it("maps a 422 empty-title error to an ApiClientError with the server's code/message", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse(
          { error: { code: "task.title_required", message: "task title is required" } },
          422,
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.createTask({ board_id: "board-1", title: "" })).rejects.toMatchObject({
      name: "ApiClientError",
      code: "task.title_required",
      message: "task title is required",
      statusCode: 422,
    });
  });
});

describe("boardApi.editTask", () => {
  it("PATCHes title/assignee to /api/v1/tasks/{taskId}", async () => {
    const updated = {
      id: "task-1",
      column_id: "col-1",
      title: "Updated title",
      assignee: null,
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:01:00Z",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(updated));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.editTask("task-1", { title: "Updated title" });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks/task-1`);
    expect(options.method).toBe("PATCH");
    expect(JSON.parse(options.body)).toEqual({ title: "Updated title" });
    expect(result).toEqual(updated);
  });

  it("maps a 404 task-not-found error to an ApiClientError", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse({ error: { code: "task.not_found", message: "task not found" } }, 404),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.editTask("missing-task", { title: "x" })).rejects.toBeInstanceOf(
      ApiClientError,
    );
  });
});

describe("boardApi.moveTask", () => {
  // Re-review 2026-08-21 re2 #4: the contract (openapi.yaml) has no
  // Idempotency-Key on move, and the API's CORS AllowedHeaders doesn't know
  // it — sending it failed the preflight and broke move in dev (:5173 →
  // :8080). A move retry is naturally idempotent server-side; the client
  // must send only contract headers.
  it("POSTs the target column to /api/v1/tasks/{taskId}/move without an Idempotency-Key header", async () => {
    const moved = {
      id: "task-1",
      column_id: "col-2",
      title: "Test task",
      assignee: null,
      created_at: "2026-08-20T00:00:00Z",
      updated_at: "2026-08-20T00:02:00Z",
    };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(moved));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.moveTask("task-1", "col-2");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks/task-1/move`);
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({ column_id: "col-2" });

    const headers = options.headers as Record<string, string>;
    expect(Object.keys(headers)).not.toContain("Idempotency-Key");

    expect(result).toEqual(moved);
  });

  it("maps a 422 invalid-column error to an ApiClientError", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse(
          { error: { code: "board.column_not_found", message: "target column does not exist" } },
          422,
        ),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.moveTask("task-1", "not-a-column")).rejects.toMatchObject({
      code: "board.column_not_found",
    });
  });
});

describe("boardApi.deleteTask", () => {
  it("DELETEs /api/v1/tasks/{taskId} and resolves with no content", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue({ ok: true, status: 204, json: () => Promise.resolve(undefined) });
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.deleteTask("task-1");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks/task-1`);
    expect(options.method).toBe("DELETE");
    expect(result).toBeUndefined();
  });

  it("maps a 404 task-not-found error to an ApiClientError", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        jsonResponse({ error: { code: "task.not_found", message: "task not found" } }, 404),
      );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.deleteTask("missing-task")).rejects.toBeInstanceOf(ApiClientError);
  });
});

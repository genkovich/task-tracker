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
  it("GETs the board state from /api/v1/board", async () => {
    const board = { columns: [], public_link: null };
    const fetchMock = vi.fn().mockResolvedValue(jsonResponse(board));
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.getBoard();

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/board`);
    expect(options?.method ?? "GET").toBe("GET");
    expect(result).toEqual(board);
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
  it("POSTs the task title/assignee to /api/v1/tasks", async () => {
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

    const result = await boardApi.createTask({ title: "Test task", assignee: "Test User" });

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks`);
    expect(options.method).toBe("POST");
    expect(JSON.parse(options.body)).toEqual({ title: "Test task", assignee: "Test User" });
    expect(result).toEqual(created);
  });

  it("maps a 422 empty-title error to an ApiClientError with the server's code/message", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse(
        { error: { code: "task.title_required", message: "task title is required" } },
        422,
      ),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.createTask({ title: "" })).rejects.toMatchObject({
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
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ error: { code: "task.not_found", message: "task not found" } }, 404),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.editTask("missing-task", { title: "x" })).rejects.toBeInstanceOf(
      ApiClientError,
    );
  });
});

describe("boardApi.moveTask", () => {
  it("POSTs the target column to /api/v1/tasks/{taskId}/move with a generated Idempotency-Key header", async () => {
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
    const idemKey = headers["Idempotency-Key"];
    expect(idemKey).toMatch(/^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$/i);

    expect(result).toEqual(moved);
  });

  it("generates a fresh Idempotency-Key on each call", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({
        id: "task-1",
        column_id: "col-2",
        title: "t",
        assignee: null,
        created_at: "2026-08-20T00:00:00Z",
        updated_at: "2026-08-20T00:00:00Z",
      }),
    );
    vi.stubGlobal("fetch", fetchMock);

    await boardApi.moveTask("task-1", "col-2");
    await boardApi.moveTask("task-1", "col-3");

    const key1 = (fetchMock.mock.calls[0][1].headers as Record<string, string>)["Idempotency-Key"];
    const key2 = (fetchMock.mock.calls[1][1].headers as Record<string, string>)["Idempotency-Key"];
    expect(key1).not.toBe(key2);
  });

  it("maps a 422 invalid-column error to an ApiClientError", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
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
    const fetchMock = vi.fn().mockResolvedValue({ ok: true, status: 204, json: () => Promise.resolve(undefined) });
    vi.stubGlobal("fetch", fetchMock);

    const result = await boardApi.deleteTask("task-1");

    expect(fetchMock).toHaveBeenCalledTimes(1);
    const [url, options] = fetchMock.mock.calls[0];
    expect(url).toBe(`${BASE_URL}/api/v1/tasks/task-1`);
    expect(options.method).toBe("DELETE");
    expect(result).toBeUndefined();
  });

  it("maps a 404 task-not-found error to an ApiClientError", async () => {
    const fetchMock = vi.fn().mockResolvedValue(
      jsonResponse({ error: { code: "task.not_found", message: "task not found" } }, 404),
    );
    vi.stubGlobal("fetch", fetchMock);

    await expect(boardApi.deleteTask("missing-task")).rejects.toBeInstanceOf(ApiClientError);
  });
});

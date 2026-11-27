import { describe, it, expect } from "vitest";

import { readFromUrl } from "./use-data-table-url-state";

const params = (entries: [string, string][]) => {
  const searchParams = new URLSearchParams();
  for (const [key, value] of entries) {
    searchParams.append(key, value);
  }
  return searchParams;
};

describe("readFromUrl", () => {
  it("derives empty state from a blank URL", () => {
    const state = readFromUrl(params([]), {}, []);
    expect(state).toEqual({ search: "", sorting: [], filters: {} });
  });

  it("parses search, sort and order", () => {
    const state = readFromUrl(
      params([["search", "ashley"], ["sort", "created_at"], ["order", "desc"]]),
      {},
      [],
    );
    expect(state.search).toBe("ashley");
    expect(state.sorting).toEqual([{ id: "created_at", desc: true }]);
  });

  it("parses single-select filter values", () => {
    const state = readFromUrl(
      params([["status", "active"]]),
      { status: [] },
      [],
    );
    expect(state.filters).toEqual({ status: "active" });
  });

  it("parses multi-select filter values", () => {
    const state = readFromUrl(
      params([
        ["status", "active"],
        ["status", "inactive"],
      ]),
      { status: [] },
      [],
    );
    expect(state.filters).toEqual({ status: ["active", "inactive"] });
  });

  it("ignores unknown params and reserved keys as filters", () => {
    const state = readFromUrl(
      params([["foo", "bar"], ["search", "x"], ["page", "2"]]),
      { status: [] },
      [],
    );
    expect(state.filters).toEqual({ status: [] });
    expect(state.search).toBe("x");
  });

  it("falls back to initial sorting when no sort param present", () => {
    const state = readFromUrl(params([]), {}, [{ id: "name", desc: false }]);
    expect(state.sorting).toEqual([{ id: "name", desc: false }]);
  });
});

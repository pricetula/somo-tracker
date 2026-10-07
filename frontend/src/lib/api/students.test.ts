import { listStudents } from "./students";

// NOTE: This is a skeleton test. In CI, mock the api client.
// Example:
// vi.mock("./client", () => ({ api: { get: vi.fn() } }));

describe("listStudents", () => {
    it("builds correct query string", async () => {
        // TODO: mock api.get and assert URL
        expect(listStudents).toBeDefined();
    });
});

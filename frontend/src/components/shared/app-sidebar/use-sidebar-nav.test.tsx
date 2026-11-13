import { describe, it, expect } from "vitest";
import { useSidebarNav, getSidebarLinksByRole } from "./use-sidebar-nav";

describe("getSidebarLinksByRole", () => {
  it("returns all links for headteacher", () => {
    const links = getSidebarLinksByRole("HEADTEACHER");
    expect(links.length).toBeGreaterThan(0);
    expect(links.some((l) => l.id === "staff")).toBe(true);
    expect(links.some((l) => l.id === "reports")).toBe(true);
    expect(links.some((l) => l.id === "fees")).toBe(true);
  });

  it("returns limited links for teacher", () => {
    const links = getSidebarLinksByRole("TEACHER");
    const ids = links.map((l) => l.id);
    expect(ids).toContain("dashboard");
    expect(ids).toContain("students");
    expect(ids).toContain("classes");
    expect(ids).toContain("guardians");
    expect(ids).not.toContain("staff");
    expect(ids).not.toContain("reports");
    expect(ids).not.toContain("fees");
  });

  it("returns limited links for guardian", () => {
    const links = getSidebarLinksByRole("GUARDIAN");
    const ids = links.map((l) => l.id);
    expect(ids).toContain("dashboard");
    expect(ids).toContain("students");
    expect(ids).toContain("profile");
    expect(ids).toContain("logout");
    expect(ids).not.toContain("classes");
    expect(ids).not.toContain("staff");
    expect(ids).not.toContain("fees");
    expect(ids.length).toBe(4);
  });

  it("returns finance links for accountant", () => {
    const links = getSidebarLinksByRole("ACCOUNTANT");
    const ids = links.map((l) => l.id);
    expect(ids).toContain("dashboard");
    expect(ids).toContain("students");
    expect(ids).toContain("fees");
    expect(ids).toContain("profile");
    expect(ids).toContain("logout");
    expect(ids).not.toContain("classes");
  });

  it("returns empty array for unknown role", () => {
    const links = getSidebarLinksByRole("UNKNOWN_ROLE");
    expect(links).toEqual([]);
  });

  it("includes logout and profile for all roles", () => {
    for (const role of ["HEADTEACHER", "TEACHER", "GUARDIAN", "ACCOUNTANT"]) {
      const ids = getSidebarLinksByRole(role).map((l) => l.id);
      expect(ids).toContain("logout");
      expect(ids).toContain("profile");
    }
  });
});

describe("useSidebarNav", () => {
  it("filters links by role from me object", () => {
    // useSidebarNav is a thin wrapper over getSidebarLinksByRole
    const headteacherLinks = useSidebarNav({ role: "HEADTEACHER" });
    expect(headteacherLinks.some((l) => l.id === "staff")).toBe(true);

    const guardianLinks = useSidebarNav({ role: "GUARDIAN" });
    expect(guardianLinks.some((l) => l.id === "staff")).toBe(false);
  });

  it("returns empty when no me provided", () => {
    const links = useSidebarNav(null);
    expect(links).toEqual([]);
  });

  it("returns empty when me has no role", () => {
    const links = useSidebarNav({} as { role?: string });
    expect(links).toEqual([]);
  });
});

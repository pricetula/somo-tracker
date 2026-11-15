import { describe, it, expect } from "vitest";
import { useSidebarNav, getSidebarLinksByRole } from "./use-sidebar-nav";
import type { MeResponse, UserRole } from "@/features/auth/types";

describe("getSidebarLinksByRole", () => {
  it("returns all links for headteacher", () => {
    const links = getSidebarLinksByRole("HEADTEACHER");
    expect(links.length).toBeGreaterThan(0);
    expect(links.some((l) => l.id === "staff")).toBe(true);
    expect(links.some((l) => l.id === "reports")).toBe(true);
    expect(links.some((l) => l.id === "fees")).toBe(true);
    expect(links.some((l) => l.id === "schools")).toBe(true);
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
    expect(ids).not.toContain("schools");
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
    expect(ids).not.toContain("schools");
    expect(ids.length).toBe(4);
  });

  it("returns finance links for finance role", () => {
    const links = getSidebarLinksByRole("FINANCE");
    const ids = links.map((l) => l.id);
    expect(ids).toContain("dashboard");
    expect(ids).toContain("students");
    expect(ids).toContain("fees");
    expect(ids).toContain("profile");
    expect(ids).toContain("logout");
    expect(ids).not.toContain("classes");
    expect(ids).not.toContain("schools");
  });

  it("returns schools link for owner", () => {
    const links = getSidebarLinksByRole("OWNER");
    expect(links.some((l) => l.id === "schools")).toBe(true);
  });

  it("returns schools link for admin", () => {
    const links = getSidebarLinksByRole("ADMIN");
    expect(links.some((l) => l.id === "schools")).toBe(true);
  });

  it("returns schools link for headteacher", () => {
    const links = getSidebarLinksByRole("HEADTEACHER");
    expect(links.some((l) => l.id === "schools")).toBe(true);
  });

  it("returns schools link for platformoperator", () => {
    const links = getSidebarLinksByRole("PLATFORMOPERATOR");
    expect(links.some((l) => l.id === "schools")).toBe(true);
  });

  it("returns empty array for unknown role", () => {
    const links = getSidebarLinksByRole("UNKNOWN_ROLE" as UserRole);
    expect(links).toEqual([]);
  });

  it("includes logout and profile for all roles", () => {
    for (const role of ["HEADTEACHER", "TEACHER", "GUARDIAN", "FINANCE"] as UserRole[]) {
      const ids = getSidebarLinksByRole(role).map((l) => l.id);
      expect(ids).toContain("logout");
      expect(ids).toContain("profile");
    }
  });
});

describe("useSidebarNav", () => {
  it("filters links by role from me object", () => {
    // useSidebarNav is a thin wrapper over getSidebarLinksByRole
    const headteacherLinks = useSidebarNav({ role: "HEADTEACHER" } as MeResponse);
    expect(headteacherLinks.some((l) => l.id === "staff")).toBe(true);
    expect(headteacherLinks.some((l) => l.id === "schools")).toBe(true);

    const guardianLinks = useSidebarNav({ role: "GUARDIAN" } as MeResponse);
    expect(guardianLinks.some((l) => l.id === "staff")).toBe(false);
    expect(guardianLinks.some((l) => l.id === "schools")).toBe(false);
  });

  it("returns empty when no me provided", () => {
    const links = useSidebarNav(null);
    expect(links).toEqual([]);
  });

  it("returns empty when me has no role", () => {
    const links = useSidebarNav({} as MeResponse);
    expect(links).toEqual([]);
  });
});

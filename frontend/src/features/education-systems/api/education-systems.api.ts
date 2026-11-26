import type {
  EducationSystem,
  EducationSystemListResponse,
  EducationSystemListItemResponse,
} from "../types/education-systems";

/**
 * listEducationSystems fetches all education systems from the backend via the BFF proxy.
 * Plain async function, no React.
 */
export async function listEducationSystems(): Promise<EducationSystem[]> {
  const res = await fetch("/api/education/systems", {
    method: "GET",
    credentials: "include",
  });

  if (!res.ok) {
    throw new Error("Failed to fetch education systems");
  }

  const raw = (await res.json()) as EducationSystemListResponse;
  return (raw.systems ?? []).map((s: EducationSystemListItemResponse) => ({
    id: s.id ?? "",
    name: s.name ?? "",
    countryId: s.country_id ?? "",
    countryName: s.country_name ?? "",
    countryCode: s.country_code ?? "",
  }));
}

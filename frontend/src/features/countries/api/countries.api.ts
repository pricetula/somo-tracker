import type { Country, CountryListResponse, CountryResponse } from "../types/countries";

/**
 * listCountries fetches all countries from the backend via the BFF proxy.
 * Plain async function, no React.
 */
export async function listCountries(): Promise<Country[]> {
  const res = await fetch("/api/education/countries", {
    method: "GET",
    credentials: "include",
  });

  if (!res.ok) {
    throw new Error("Failed to fetch countries");
  }

  const raw = (await res.json()) as CountryListResponse;
  return (raw.countries ?? []).map((c: CountryResponse) => ({
    id: c.id ?? "",
    name: c.name ?? "",
    code: c.code ?? "",
  }));
}

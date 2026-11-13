export const countryKeys = {
  all: ['country'] as const,
  list: () => [...countryKeys.all, 'list'] as const,
};

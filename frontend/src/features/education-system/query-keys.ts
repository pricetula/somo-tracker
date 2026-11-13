export const educationKeys = {
  all: ['education-system'] as const,
  list: (countryId: string) => [...educationKeys.all, 'list', countryId] as const,
};

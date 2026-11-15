export const schoolKeys = {
  all: ['schools'] as const,
  list: () => [...schoolKeys.all, 'list'] as const,
  detail: (id: string) => [...schoolKeys.all, 'detail', id] as const,
};

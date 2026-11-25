export const educationSystemKeys = {
  all: ["education-systems"] as const,
  list: () => [...educationSystemKeys.all, "list"] as const,
};

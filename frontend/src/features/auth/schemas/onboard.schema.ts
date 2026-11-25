import { z } from "zod";

export const onboardSchema = z.object({
  organizationName: z.string().min(1, "Organization name is required"),
  schoolName: z.string().min(1, "School name is required"),
  userName: z.string().min(1, "Your name is required"),
  countryId: z.string().min(1, "Country is required"),
  educationSystemId: z.string().min(1, "Education system is required"),
});

export type OnboardFormValues = z.infer<typeof onboardSchema>;

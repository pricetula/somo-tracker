"use client";

import { Suspense } from "react";
import { useSearchParams, useRouter } from "next/navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useEffect } from "react";
import { Loader2, GraduationCap } from "lucide-react";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
  FormDescription,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useRegister } from "@/hooks/use-auth";
import { EducationSystemCombobox } from "@/features/education-system";
import { DocTooltip } from "@/components/ui/DocTooltip";

interface RegisterPageProps {
  tooltipSummary?: string;
}

const registerSchema = z.object({
  school_name: z
    .string()
    .min(2, "School name must be at least 2 characters")
    .max(100, "School name must be less than 100 characters"),
  first_name: z
    .string()
    .min(1, "First name is required")
    .max(100, "First name must be less than 100 characters"),
  last_name: z
    .string()
    .min(1, "Last name is required")
    .max(100, "Last name must be less than 100 characters"),
  education_system_id: z
    .string()
    .uuid("Please select an education system"),
});

type RegisterValues = z.infer<typeof registerSchema>;

function RegisterForm({ tooltipSummary }: { tooltipSummary?: string }) {
  const searchParams = useSearchParams();
  const router = useRouter();
  const sessionRef = searchParams.get("session_ref");
  const registerMutation = useRegister();

  const form = useForm<RegisterValues>({
    resolver: zodResolver(registerSchema),
    defaultValues: {
      school_name: "",
      first_name: "",
      last_name: "",
      education_system_id: "",
    },
  });

  // Redirect to login if no session_ref is present
  useEffect(() => {
    if (!sessionRef) {
      router.replace("/login");
    }
  }, [sessionRef, router]);

  if (!sessionRef) {
    return null;
  }

  function onSubmit(values: RegisterValues) {
    registerMutation.mutate({
      school_name: values.school_name,
      session_ref: sessionRef!,
      first_name: values.first_name,
      last_name: values.last_name,
      education_system_id: values.education_system_id,
    });
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-md">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Create Your School Account</CardTitle>
          <CardDescription>
            Set up your school or educational organization
            {tooltipSummary && (
              <DocTooltip summary={tooltipSummary} slug="authentication" />
            )}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="school_name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>School Name</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="e.g. Lincoln High School"
                        autoFocus
                        {...field}
                      />
                    </FormControl>
                    <FormDescription>
                      The name of your educational institution
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="grid grid-cols-2 gap-3">
                <FormField
                  control={form.control}
                  name="first_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>First Name</FormLabel>
                      <FormControl>
                        <Input placeholder="Jane" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="last_name"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Last Name</FormLabel>
                      <FormControl>
                        <Input placeholder="Doe" {...field} />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
              <FormField
                control={form.control}
                name="education_system_id"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Education System</FormLabel>
                    <FormControl>
                      <EducationSystemCombobox
                        value={field.value}
                        onValueChange={field.onChange}
                        placeholder="Select education system…"
                      />
                    </FormControl>
                    <FormDescription>
                      The curriculum framework your school follows
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <div className="flex items-center gap-2 rounded-xl border border-border/50 bg-muted/30 px-4 py-3 text-sm text-muted-foreground">
                <GraduationCap className="h-4 w-4 shrink-0" />
                <span>
                  Your school&apos;s education system determines available
                  curricula, grade structure, and assessment types.
                </span>
              </div>
              <Button
                type="submit"
                className="w-full"
                disabled={registerMutation.isPending}
              >
                {registerMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Create Account
              </Button>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}

export function RegisterPage({ tooltipSummary }: RegisterPageProps) {
  return (
    <Suspense
      fallback={
        <div className="flex min-h-screen items-center justify-center">
          <Loader2 className="h-8 w-8 animate-spin text-muted-foreground" />
        </div>
      }
    >
      <RegisterForm tooltipSummary={tooltipSummary} />
    </Suspense>
  );
}

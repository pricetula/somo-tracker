"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { Button } from "@/components/ui/button";
import {
  Form,
  FormControl,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { useDiscover } from "@/hooks/use-auth";
import { Loader2 } from "lucide-react";
import { DocTooltip } from "@/components/ui/DocTooltip";

interface LoginPageProps {
  tooltipSummary?: string;
}

const loginSchema = z.object({
  email: z.string().email("Please enter a valid email address"),
});

type LoginValues = z.infer<typeof loginSchema>;

export function LoginPage({ tooltipSummary }: LoginPageProps) {
  const discoverMutation = useDiscover();

  const form = useForm<LoginValues>({
    resolver: zodResolver(loginSchema),
    defaultValues: {
      email: "",
    },
  });

  function onSubmit(values: LoginValues) {
    discoverMutation.mutate(values.email);
  }

  return (
    <div className="flex min-h-screen items-center justify-center px-4">
      <Card className="w-full max-w-sm">
        <CardHeader className="text-center">
          <CardTitle className="text-2xl">Sign in to Somotracker</CardTitle>
          <CardDescription>
            Enter your email to receive a magic link
          </CardDescription>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
              <FormField
                control={form.control}
                name="email"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Email</FormLabel>
                    <FormControl>
                      <Input
                        type="email"
                        placeholder="you@school.edu"
                        autoComplete="email"
                        autoFocus
                        {...field}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <Button
                type="submit"
                className="w-full"
                disabled={discoverMutation.isPending}
              >
                {discoverMutation.isPending && (
                  <Loader2 className="mr-2 h-4 w-4 animate-spin" />
                )}
                Send Magic Link
              </Button>
            </form>
          </Form>
          <p className="mt-4 text-center text-xs text-muted-foreground">
            No password needed. We&apos;ll send a one-time sign-in link to your
            email.
            {tooltipSummary && (
              <DocTooltip summary={tooltipSummary} slug="authentication" anchorId="session-timeouts" />
            )}
          </p>
        </CardContent>
      </Card>
    </div>
  );
}

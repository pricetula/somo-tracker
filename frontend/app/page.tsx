export default function Page() {
  return (
    <main className="flex min-h-screen flex-col items-center justify-center p-8">
      <h1 className="text-4xl font-bold tracking-tight">Somotracker</h1>
      <p className="mt-4 text-muted-foreground">
        School management system for Kenyan schools (PP1–Grade 9, CBC)
      </p>
      <div className="mt-8 flex gap-4">
        <a
          href="/api/healthz"
          className="rounded-md bg-primary px-4 py-2 text-primary-foreground hover:bg-primary/90"
        >
          Health Check
        </a>
        <a
          href="/api/readyz"
          className="rounded-md border border-input bg-background px-4 py-2 hover:bg-accent"
        >
          Readiness Check
        </a>
      </div>
    </main>
  );
}
import { createFileRoute } from '@tanstack/react-router';

export const Route = createFileRoute('/')({
  component: Home,
});

function Home() {
  return (
    <div className="space-y-4">
      <h1 className="text-3xl font-bold">Somotracker</h1>
      <p className="text-muted-foreground">TanStack Start scaffold ready.</p>
    </div>
  );
}

import { createFileRoute } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';

export const Route = createFileRoute('/')({
  component: Home,
});

export function Home() {
  return (
    <div className="space-y-4">
      <h1 className="text-3xl font-bold">Somotracker</h1>
      <p className="text-muted-foreground">TanStack Start scaffold ready.</p>
      <Button variant="default" size="default">
        Test Button
      </Button>
    </div>
  );
}

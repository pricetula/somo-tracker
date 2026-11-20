import { LogoutButton } from '@/components/logout-button'

export default function RootPage() {
  return (
    <div className="flex min-h-screen flex-col items-center justify-center p-4">
      <div className="w-full max-w-md space-y-6">
        <h1 className="text-3xl font-bold">Dashboard</h1>
        <p className="text-muted-foreground">Welcome to Somotracker.</p>
        <LogoutButton />
      </div>
    </div>
  );
}

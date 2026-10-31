/**
 * Unauthorized route - 403 page
 */

import { createFileRoute } from '@tanstack/react-router';
import { Link } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from '@/components/ui/card';
import { Alert, AlertDescription } from '@/components/ui/alert';
import { AlertCircle, Lock } from 'lucide-react';

export const Route = createFileRoute('/unauthorized')({
  component: UnauthorizedPage,
});

function UnauthorizedPage() {
  return (
    <div className="min-h-screen flex items-center justify-center bg-background px-4 py-12">
      <Card className="w-full max-w-md mx-auto">
        <CardHeader className="space-y-4 text-center">
          <div className="mx-auto h-12 w-12 rounded-lg bg-destructive/10 flex items-center justify-center">
            <Lock className="h-6 w-6 text-destructive" />
          </div>
          <CardTitle className="text-2xl">Access Denied</CardTitle>
          <CardDescription>
            You don&apos;t have permission to access this resource
          </CardDescription>
        </CardHeader>
        <CardContent className="space-y-4">
          <Alert variant="destructive">
            <AlertCircle className="h-4 w-4" />
            <AlertDescription>
              Your account doesn&apos;t have the required role for this area. Contact your administrator if you believe this is an error.
            </AlertDescription>
          </Alert>

          <div className="space-y-3">
            <Link to="/" className="w-full">
              <Button variant="default" className="w-full">
                Go to Dashboard
              </Button>
            </Link>
            <Link to="/login" className="w-full">
              <Button variant="outline" className="w-full">
                Sign in as different user
              </Button>
            </Link>
          </div>
        </CardContent>
      </Card>
    </div>
  );
}
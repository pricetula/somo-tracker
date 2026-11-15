import { createFileRoute, useParams, useNavigate } from '@tanstack/react-router';
import { Button } from '@/components/ui/button';
import { toast } from 'sonner';
import { getSchool, deactivateSchools, type School } from '@/features/schools';
import { useState } from 'react';
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetFooter,
  SheetTitle,
  SheetDescription,
} from '@/components/ui/sheet';

// This @modal route returns null on direct/hard load.
// It is rendered as a sheet from the parent (schools/index.tsx)
// when the user clicks a school row via client-side navigation.
export const Route = createFileRoute('/schools/@modal/$schoolId')({
  component: () => null,
});

export function SchoolDetailSheet({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const { schoolId } = useParams();
  const navigate = useNavigate();
  const [isDeleting, setIsDeleting] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  const [school, setSchool] = useState<School | null>(null);

  const loadSchool = async () => {
    if (!schoolId) return;
    try {
      const schoolData = await getSchool(schoolId);
      setSchool(schoolData);
    } catch (error) {
      toast.error('Failed to load school');
    }
  };

  const handleDelete = async () => {
    if (!schoolId) return;
    try {
      await deactivateSchools({ ids: [schoolId] });
      toast.success('School deactivated');
      onClose();
      navigate({ to: '/schools' });
    } catch (error) {
      toast.error('Failed to deactivate school');
    }
  };

  if (!school) {
    return null;
  }

  return (
    <Sheet open={open} onOpenChange={(o) => (o ? null : onClose())}>
      <SheetContent side="right" className="w-96">
        <SheetHeader>
          <SheetTitle>{school.name}</SheetTitle>
          <SheetDescription>{school.code}</SheetDescription>
        </SheetHeader>
        <div className="p-6 space-y-4">
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground uppercase tracking-wider">ID</p>
            <p className="font-mono text-sm">{school.id}</p>
          </div>
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground uppercase tracking-wider">Tenant ID</p>
            <p className="font-mono text-sm">{school.tenantId}</p>
          </div>
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground uppercase tracking-wider">Country</p>
            <p className="text-sm">{school.countryId}</p>
          </div>
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground uppercase tracking-wider">Education System</p>
            <p className="text-sm">{school.educationSystemId}</p>
          </div>
          <div className="space-y-2">
            <p className="text-xs text-muted-foreground uppercase tracking-wider">Status</p>
            <p className={school.deletedAt ? 'text-sm text-red-900' : 'text-sm text-green-900'}>
              {school.deletedAt ? 'Deactivated' : 'Active'}
            </p>
          </div>
        </div>
        <SheetFooter>
          <div className="space-y-2">
            <Button
              variant="destructive"
              onClick={() => setConfirmDeleteOpen(true)}
              disabled={isDeleting}
            >
              Deactivate School
            </Button>
            <Button variant="outline" onClick={onClose}>
              Close
            </Button>
          </div>
        </SheetFooter>
      </SheetContent>

      {confirmDeleteOpen && (
        <AlertDialog open={confirmDeleteOpen} onOpenChange={setConfirmDeleteOpen}>
          <AlertDialogContent>
            <AlertDialogHeader>
              <AlertDialogTitle>Deactivate School</AlertDialogTitle>
            </AlertDialogHeader>
            <AlertDialogDescription>
              Are you sure you want to deactivate {school.name}?
            </AlertDialogDescription>
            <AlertDialogFooter>
              <AlertDialogCancel>Cancel</AlertDialogCancel>
              <AlertDialogAction onClick={handleDelete} variant="destructive">
                Deactivate
              </AlertDialogAction>
            </AlertDialogFooter>
          </AlertDialogContent>
        </AlertDialog>
      )}
    </Sheet>
  );
}

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

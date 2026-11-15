import { createFileRoute, Link, useParams } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { getSchool, updateSchool, deactivateSchools, type School } from '@/features/schools';
import { toast } from 'sonner';
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

const updateSchoolSchema = z.object({
  name: z.string().min(1, 'School name is required'),
  code: z.string().min(1, 'School code is required'),
  countryId: z.string().min(1, 'Country is required'),
  educationSystemId: z.string().min(1, 'Education system is required'),
});

type UpdateSchoolFormData = z.infer<typeof updateSchoolSchema>;

export const Route = createFileRoute('/$schoolId')({
  component: SchoolPage,
});

function SchoolPage() {
  const { schoolId } = useParams();
  const [isDeleting, setIsDeleting] = useState(false);
  const [confirmDeleteOpen, setConfirmDeleteOpen] = useState(false);

  const navigate = (to: string) => {
    window.location.href = to;
  };

  const [school, setSchool] = useState<School | null>(null);
  const form = useForm<UpdateSchoolFormData>({
    resolver: zodResolver(updateSchoolSchema),
    defaultValues: {
      name: '',
      code: '',
      countryId: '',
      educationSystemId: '',
    },
  });

  const loadSchool = async () => {
    if (!schoolId) return;
    try {
      const schoolData = await getSchool(schoolId);
      setSchool(schoolData);
      form.reset({
        name: schoolData.name,
        code: schoolData.code,
        countryId: schoolData.countryId,
        educationSystemId: schoolData.educationSystemId,
      });
    } catch (error) {
      toast.error('Failed to load school');
    }
  };

  const onSubmit = async (data: UpdateSchoolFormData) => {
    if (!schoolId) return;
    try {
      await updateSchool(schoolId, data);
      toast.success('School updated successfully');
      const updated = await getSchool(schoolId);
      setSchool(updated);
      form.reset({
        name: updated.name,
        code: updated.code,
        countryId: updated.countryId,
        educationSystemId: updated.educationSystemId,
      });
    } catch (error) {
      toast.error('Failed to update school');
    }
  };

  const handleDelete = async () => {
    if (!schoolId) return;
    try {
      await deactivateSchools({ ids: [schoolId] });
      toast.success('School deactivated');
      navigate('/schools');
    } catch (error) {
      toast.error('Failed to deactivate school');
    }
  };

  if (!school) {
    return (
      <div className="p-6 flex flex-col items-center justify-center min-h-[400px]">
        <div className="animate-pulse rounded-lg bg-muted w-96 h-32" />
      </div>
    );
  }

  return (
    <div className="p-6">
      <div className="flex flex-col space-y-1 pb-6">
        <h1 className="text-2xl font-bold">{school.name}</h1>
        <p className="text-sm text-muted-foreground">{school.code}</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">ID</p>
          <p className="font-mono text-sm">{school.id}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Tenant ID</p>
          <p className="font-mono text-sm">{school.tenantId}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Country</p>
          <p className="text-sm">{school.countryId}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Education System</p>
          <p className="text-sm">{school.educationSystemId}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Created</p>
          <p className="text-sm">{new Date(school.createdAt).toLocaleDateString()}</p>
        </div>
        <div className="p-4 bg-muted rounded-lg">
          <p className="text-xs text-muted-foreground uppercase tracking-wider">Status</p>
          <p className={school.deletedAt ? 'text-sm text-red-900' : 'text-sm text-green-900'}>
            {school.deletedAt ? 'Deactivated' : 'Active'}
          </p>
        </div>
      </div>

      <div className="mt-8">
        <h2 className="text-lg font-bold mb-4">Edit School</h2>
        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-4">
            <FormField name="name">
              {({ field }: any) => (
                <FormItem>
                  <FormLabel>School Name</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="Enter school name" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            </FormField>
            <FormField name="code">
              {({ field }: any) => (
                <FormItem>
                  <FormLabel>School Code</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="Enter school code" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            </FormField>
            <FormField name="countryId">
              {({ field }: any) => (
                <FormItem>
                  <FormLabel>Country</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="Enter country ID" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            </FormField>
            <FormField name="educationSystemId">
              {({ field }: any) => (
                <FormItem>
                  <FormLabel>Education System</FormLabel>
                  <FormControl>
                    <Input {...field} placeholder="Enter education system ID" />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            </FormField>
            <div className="flex justify-end space-y-2">
              <Button type="button" variant="outline" onClick={() => form.reset({
                name: school.name,
                code: school.code,
                countryId: school.countryId,
                educationSystemId: school.educationSystemId,
              })}>
                Cancel
              </Button>
              <Button type="submit">Save Changes</Button>
            </div>
          </form>
        </Form>
      </div>

      <div className="mt-8">
        <Button variant="destructive" onClick={() => setConfirmDeleteOpen(true)}>
          Deactivate School
        </Button>
      </div>

      <AlertDialog open={confirmDeleteOpen} onOpenChange={setConfirmDeleteOpen}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>Deactivate School</AlertDialogTitle>
            <AlertDialogDescription>
              Are you sure you want to deactivate "{school.name}"? This cannot be undone.
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>Cancel</AlertDialogCancel>
            <AlertDialogAction onClick={handleDelete}>Deactivate</AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

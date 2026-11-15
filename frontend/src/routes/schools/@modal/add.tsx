import { createFileRoute, useNavigate } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { createSchool } from '@/features/schools';
import { toast } from 'sonner';
import {
  AlertDialog,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';

const createSchoolSchema = z.object({
  name: z.string().min(1, 'School name is required'),
  code: z.string().min(1, 'School code is required'),
  countryId: z.string().min(1, 'Country is required'),
  educationSystemId: z.string().min(1, 'Education system is required'),
});

type CreateSchoolFormData = z.infer<typeof createSchoolSchema>;

// This @modal route returns null on direct/hard load.
// It is rendered as a dialog from the parent (schools/index.tsx)
// when the user clicks "Add School" via client-side navigation.
export const Route = createFileRoute('/schools/@modal/add')({
  component: () => null,
});

export function AddSchoolDialog({
  open,
  onClose,
}: {
  open: boolean;
  onClose: () => void;
}) {
  const navigate = useNavigate();
  const form = useForm<CreateSchoolFormData>({
    resolver: zodResolver(createSchoolSchema),
    defaultValues: {
      name: '',
      code: '',
      countryId: '',
      educationSystemId: '',
    },
  });

  const onSubmit = async (data: CreateSchoolFormData) => {
    try {
      await createSchool(data);
      toast.success('School created successfully');
      onClose();
      navigate({ to: '/schools' });
    } catch (error) {
      toast.error('Failed to create school');
    }
  };

  return (
    <AlertDialog open={open} onOpenChange={(o) => (o ? null : onClose())}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>Add School</AlertDialogTitle>
        </AlertDialogHeader>
        <AlertDialogDescription>
          Create a new school in your tenant.
        </AlertDialogDescription>
        <AlertDialogFooter>
          <Form {...form}>
            <form onSubmit={form.handleSubmit(onSubmit)} className="space-y-6">
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
              <div className="flex justify-end space-x-2">
                <Button type="button" variant="outline" onClick={onClose}>
                  Cancel
                </Button>
                <Button type="submit">Create School</Button>
              </div>
            </form>
          </Form>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  );
}

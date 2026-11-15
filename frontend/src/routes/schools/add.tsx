import { createFileRoute, Link } from '@tanstack/react-router';
import { useForm } from 'react-hook-form';
import { zodResolver } from '@hookform/resolvers/zod';
import { z } from 'zod';
import { Button } from '@/components/ui/button';
import { Input } from '@/components/ui/input';
import { Form, FormControl, FormField, FormItem, FormLabel, FormMessage } from '@/components/ui/form';
import { createSchool } from '@/features/schools';
import { toast } from 'sonner';

const createSchoolSchema = z.object({
  name: z.string().min(1, 'School name is required'),
  code: z.string().min(1, 'School code is required'),
  countryId: z.string().min(1, 'Country is required'),
  educationSystemId: z.string().min(1, 'Education system is required'),
});

type CreateSchoolFormData = z.infer<typeof createSchoolSchema>;

export const Route = createFileRoute('/schools/add')({
  component: CreateSchoolPage,
});

function CreateSchoolPage() {
  const navigate = (to: string) => {
    window.location.href = to;
  };
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
      navigate('/schools');
    } catch (error) {
      toast.error('Failed to create school');
    }
  };

  return (
    <div className="p-6">
      <div className="flex flex-col space-y-1 pb-6">
        <h1 className="text-2xl font-bold">Add School</h1>
        <p className="text-sm text-muted-foreground">Add School</p>
      </div>
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
          <div className="flex justify-end space-y-2">
            <Link to="/schools" className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md border border-input bg-input/50 hover:bg-input/80 text-foreground">
              Cancel
            </Link>
            <Button type="submit">Create School</Button>
          </div>
        </form>
      </Form>
    </div>
  );
}

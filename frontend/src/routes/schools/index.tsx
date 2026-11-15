import { createFileRoute } from '@tanstack/react-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import { useState } from 'react';
import { Button } from '@/components/ui/button';
import { schoolKeys, type School } from '@/features/schools';
import { AddSchoolDialog } from '@/routes/schools/@modal/add';
import { SchoolDetailSheet } from '@/routes/schools/@modal/$schoolId';

interface SchoolPage {
  items: School[];
  nextCursor: string | null;
}

export const Route = createFileRoute('/schools/')({
  component: SchoolsPage,
});

function SchoolsPage() {
  const [showAddDialog, setShowAddDialog] = useState(false);
  const [showSchoolSheet, setShowSchoolSheet] = useState(false);
  const [selectedSchoolId, setSelectedSchoolId] = useState<string | null>(null);

  const query = useInfiniteQuery({
    queryKey: schoolKeys.list(),
    queryFn: async (
      { pageParam }: { pageParam: string | undefined },
    ): Promise<SchoolPage> => {
      const params = new URLSearchParams();
      if (pageParam) {
        params.set('cursor', pageParam);
      }
      const res = await fetch(`/api/v1/schools?${params}`);
      return (await res.json()) as SchoolPage;
    },
    getNextPageParam: (lastPage: SchoolPage) => lastPage.nextCursor,
    initialPageParam: undefined as string | undefined,
    staleTime: 0,
    gcTime: 0,
  });

  const data = (query.data?.pages.flatMap((page) => page.items) ?? []) as School[];
  const isLoading = query.isFetching || query.isLoading;

  return (
    <div className="p-6">
      <div className="flex flex-col space-y-1 pb-6">
        <h1 className="text-2xl font-bold">Schools</h1>
        <p className="text-sm text-muted-foreground">
          Manage schools in your tenant
        </p>
        <div className="mt-4 flex items-end gap-2">
          <Button
            onClick={(e) => {
              e.preventDefault();
              setShowAddDialog(true);
            }}
          >
            Add School
          </Button>
        </div>
      </div>
      {isLoading && data.length === 0 && (
        <div className="flex items-center justify-center h-48">
          <div className="animate-pulse bg-muted rounded w-full h-8" />
        </div>
      )}
      {data.length === 0 && !isLoading && (
        <div className="flex flex-col items-center justify-center py-12 text-muted-foreground">
          <p className="text-sm">No results</p>
        </div>
      )}
      <AddSchoolDialog open={showAddDialog} onClose={() => setShowAddDialog(false)} />
      {selectedSchoolId && (
        <SchoolDetailSheet
          open={showSchoolSheet}
          onClose={() => {
            setShowSchoolSheet(false);
            setSelectedSchoolId(null);
          }}
        />
      )}
    </div>
  );
}

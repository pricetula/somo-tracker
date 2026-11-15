import { createFileRoute, Link } from '@tanstack/react-router';
import { useInfiniteQuery } from '@tanstack/react-query';
import { DataTable } from '@/components/shared/data-table';
import {
  textColumn,
  badgeColumn,
  selectColumn,
  linkColumn,
  type ColumnDef,
} from '@/components/shared/data-table/column-factories';
import {
  getSchools,
  schoolKeys,
  type School,
} from '@/features/schools';
import { createListQueryFn } from '@/components/shared/data-table/create-list-query-fn';

export const Route = createFileRoute('/schools')({
  component: SchoolsPage,
});

function SchoolsPage() {
  const query = useInfiniteQuery({
    queryKey: schoolKeys.list(),
    queryFn: createListQueryFn<School>('/api/v1/schools'),
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialPageParam: undefined as string | undefined,
    staleTime: 0,
    gcTime: 0,
  });

  const data = (query.data?.pages.flatMap((page: { items: School[] }) => page.items) ?? []) as School[];
  const nextCursor = query.data?.pages[query.data.pages.length - 1]?.nextCursor ?? null;
  const isLoading = query.isFetching || query.isLoading;
  const isFetchingNextPage = query.isFetchingNextPage;

  const columns: ColumnDef<any, any>[] = [
    selectColumn(),
    linkColumn({
      accessorKey: 'name',
      label: 'Name',
      sortKeys: ['name_asc', 'name_desc'],
      to: (row) => `/schools/${row.id}`,
      width: '1fr',
    }),
    textColumn({
      accessorKey: 'code',
      label: 'Code',
      sortKeys: ['code_asc', 'code_desc'],
      width: '120px',
    }),
    textColumn({
      accessorKey: 'countryId',
      label: 'Country',
      sortKeys: ['countryId_asc', 'countryId_desc'],
      width: '120px',
    }),
    textColumn({
      accessorKey: 'educationSystemId',
      label: 'Education System',
      sortKeys: ['educationSystemId_asc', 'educationSystemId_desc'],
      width: '150px',
    }),
    badgeColumn({
      accessorKey: 'deletedAt',
      label: 'Status',
      variants: {
        null: 'bg-green-10 text-green-900',
        'not-null': 'bg-gray-10 text-gray-900',
      },
      defaultVariant: 'bg-gray-10 text-gray-900',
      width: '100px',
    }),
  ];

  const getRowId = (index: number, school: School) => school.id;

  return (
    <div className="p-6">
      <div className="flex flex-col space-y-1 pb-6">
        <h1 className="text-2xl font-bold">Schools</h1>
        <p className="text-sm text-muted-foreground">Manage schools in your tenant</p>
        <div className="mt-4 flex items-end gap-2">
          <Link to="/schools/add" className="inline-flex items-center gap-2 px-3 py-1.5 text-xs font-medium rounded-md border border-input bg-input/50 hover:bg-input/80 text-foreground">
            Add School
          </Link>
        </div>
      </div>
      <DataTable<School>
        columns={columns}
        data={data}
        nextCursor={nextCursor}
        isLoading={isLoading}
        isFetchingNextPage={isFetchingNextPage}
        fetchNextPage={() => query.fetchNextPage()}
        onSortChange={(sort) => {
          // Sort handled by URL query params via createListQueryFn
        }}
        onFilterChange={(filters) => {
          // Filter handled by URL query params via createListQueryFn
        }}
        getRowId={getRowId}
      />
    </div>
  );
}

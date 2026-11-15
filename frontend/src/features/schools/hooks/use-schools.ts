'use client'

import { useInfiniteQuery } from '@tanstack/react-query';
import { getSchools, type GetSchoolsParams } from '../api/get-schools';
import { schoolKeys } from '../query-keys';

export function useSchools(params: GetSchoolsParams = {}) {
  return useInfiniteQuery({
    queryKey: schoolKeys.list(),
    queryFn: ({ pageParam }) => getSchools({ ...params, cursor: pageParam as string | undefined }),
    getNextPageParam: (lastPage) => lastPage.nextCursor,
    initialPageParam: undefined as string | undefined,
    staleTime: 0,
    gcTime: 0,
  });
}

// App-wide providers: TanStack Query (SWR semantics) + offline sync engine.
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { type ReactNode, useEffect, useState } from 'react';
import { startSync } from '../shared/offline/sync';

function makeQueryClient(): QueryClient {
  return new QueryClient({
    defaultOptions: {
      // networkMode 'always': our queryFns/mutationFns handle offline
      // themselves (Dexie mirrors + outbox) — never pause them (US5).
      mutations: {
        networkMode: 'always',
      },
      queries: {
        networkMode: 'always',
        staleTime: 30_000,
        retry: (failureCount, error) => {
          // Never retry auth failures; retry network blips once.
          if (error instanceof Error && 'status' in error) {
            const status = (error as { status: number }).status;
            if (status === 401 || status === 403 || status === 404) return false;
          }
          return failureCount < 1;
        },
        refetchOnWindowFocus: true,
      },
    },
  });
}

export function Providers({ children }: { children: ReactNode }) {
  const [queryClient] = useState(makeQueryClient);

  useEffect(() => startSync(queryClient), [queryClient]);

  return <QueryClientProvider client={queryClient}>{children}</QueryClientProvider>;
}
